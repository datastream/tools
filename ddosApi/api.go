package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/hashicorp/consul/api"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DDoSAPI struct {
	client      *api.Client
	Setting     map[string]string
	routeTable  map[string]string
	exitChan    chan int
	redisClient *redis.Client
	sync.RWMutex
}

func (m *DDoSAPI) Stop() {
	close(m.exitChan)
}

func (m *DDoSAPI) Run() {
	ticker := time.Tick(time.Second * 600)
	config := api.DefaultConfig()
	config.Address = m.Setting["consul_address"]
	config.Datacenter = m.Setting["datacenter"]
	config.Token = m.Setting["consul_token"]
	m.redisClient = redis.NewClient(&redis.Options{
		Addr:     m.Setting["redis_server"],
		Password: m.Setting["redis_passoword"],
		DB:       0,
	})
	_, err := m.redisClient.Ping().Result()
	if err != nil {
		fmt.Println("redis server failed", err)
	}
	m.client, err = api.NewClient(config)
	if err != nil {
		fmt.Println("reload consul setting failed", err)
	}
	err = m.ReadAPIConfig()
	for {
		select {
		case <-ticker:
			err = m.ReadAPIConfig()
			if err != nil {
				fmt.Println("reload consul setting failed", err)
			}
		case <-m.exitChan:
			return
		}
	}
}

func (m *DDoSAPI) ReadAPIConfig() error {
	l := len(m.Setting["cluster"])
	if m.Setting["cluster"][l-1] == '/' {
		m.Setting["cluster"] = m.Setting["cluster"][:l-1]
	}
	newConf, err := m.ReadConfigFromConsul(m.Setting["cluster"])
	if err != nil {
		return err
	}
	m.Lock()
	m.routeTable = newConf
	m.Unlock()
	return nil
}

func (m *DDoSAPI) ReadConfigFromConsul(key string) (map[string]string, error) {
	consulSetting := make(map[string]string)
	kv := m.client.KV()
	pairs, _, err := kv.List(key, nil)
	if err != nil {
		return consulSetting, err
	}
	size := len(key)
	for _, value := range pairs {
		if len(value.Key) > size {
			consulSetting[value.Key[size:]] = string(value.Value)
		}
	}
	return consulSetting, err
}

func (m *DDoSAPI) APIGet(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	url := c.Request.URL.Path[7:]
	url = fmt.Sprintf("%s_read", url)
	m.RLock()
	endpoints, ok := m.routeTable[url]
	m.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "bad url"})
		return
	}
	endpoints = strings.Trim(endpoints, "/")
	data, _ := m.redisClient.SMembers(endpoints).Result()
	body := make(map[string]string)
	for _, v := range data {
		key := fmt.Sprintf("%s/%s", endpoints, v)
		rst, err := m.redisClient.Get(key).Result()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "read redis error"})
			return
		}
		body[key] = rst
	}
	c.JSON(http.StatusOK, body)
}

func (m *DDoSAPI) APISet(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	url := c.Request.URL.Path[7:]
	m.RLock()
	endpoints, ok := m.routeTable[url]
	m.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "bad url"})
		return
	}
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	buf := bytes.NewBuffer(body)
	rst, err := sendrequest(endpoints, buf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, rst)
	clientIP := c.Request.Header.Get("X-From-IP")
	if len(clientIP) < 1 {
		clientIP = c.ClientIP()
	}
	logger.Printf(`"{"IP":"%s","API":"%s","Cmd":"%s"}"`, clientIP, c.Request.URL.Path, string(body))

}

func sendrequest(url string, buf io.Reader) ([]byte, error) {
	client := http.Client{}
	resp, err := client.Post(url, "application/x-www-form-urlencoded", buf)
	var body []byte
	if err == nil {
		if resp.StatusCode != 200 {
			log.Printf("unsuccessfull return %s\n", resp.Status)
		}
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	return body, err
}
