package main

import (
	"bytes"
	"fmt"
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

// APIHandle will route request to different endpoints
func (m *DDoSAPI) APIHandle(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	m.RLock()
	endpoints, ok := m.routeTable[url]
	m.RUnlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method == "GET" {
		data, err := m.ReadConfigFromConsul(fmt.Sprintf("ddosagent/status/%s", url))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
		w.WriteHeader(http.StatusOK)
		for _, v := range data {
			w.Write([]byte(v))
		}
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	buf := bytes.NewBuffer(body)
	rst, err := sendrequest(endpoints, buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	w.WriteHeader(http.StatusOK)
	w.Write(rst)
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
