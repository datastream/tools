package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type DDoSAPI struct {
	client     *api.Client
	Setting    map[string]string
	routeTable map[string]string
	exitChan   chan int
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
	var err error
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
	size := len(key) + 1
	for _, value := range pairs {
		if len(value.Key) > size {
			consulSetting[value.Key[size:]] = string(value.Value)
		}
	}
	return consulSetting, err
}

func (m *DDoSAPI) APIGet(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	url := c.Request.URL.Path
	m.RLock()
	_, ok := m.routeTable[url]
	m.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "bad url"})
		return
	}
	data, err := m.ReadConfigFromConsul(fmt.Sprintf("ddosagent/status/%s", url))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "read consule error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (m *DDoSAPI) APISet(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	url := c.Request.URL.Path
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
