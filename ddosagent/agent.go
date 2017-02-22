package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type DDoSAgent struct {
	IPSets           map[string]*IPSet
	APITasks         map[string]*APITask
	client           *api.Client
	agent            *api.Agent
	HostName         string
	Setting          map[string]string
	Port             int
	LookupdAddresses []string
	exitChan         chan int
}

func (m *DDoSAgent) Stop() {
	close(m.exitChan)
	for _, v := range m.IPSets {
		v.Stop()
	}
	for _, v := range m.APITasks {
		v.Stop()
	}
}

func (m *DDoSAgent) Run() {
	m.IPSets = make(map[string]*IPSet)
	m.APITasks = make(map[string]*APITask)
	ticker := time.Tick(time.Second * 600)
	items := strings.Split(m.Setting["listen_addr"], ":")
	if len(items) < 2 {
		log.Fatal("bad listen_addr value")
	}
	var err error
	m.Port, err = strconv.Atoi(items[len(items)-1])
	if err != nil {
		log.Fatal("get listen port failed")
	}
	m.HostName, err = os.Hostname()
	if err != nil {
		log.Fatal("get hostname failed", err)
	}
	config := api.DefaultConfig()
	config.Address = m.Setting["consul_address"]
	config.Datacenter = m.Setting["datacenter"]
	config.Token = m.Setting["consul_token"]
	m.client, err = api.NewClient(config)
	if err != nil {
		fmt.Println("reload consul setting failed", err)
	}
	m.agent = &api.Agent{}
	err = m.agent.Join(m.Setting["consul_address"], false)
	if err != nil {
		fmt.Println("join consul agent failed", err)
	}
	err = m.ReadIPSetConfig()
	if err != nil {
		fmt.Println("reload consul setting failed", err)
	}
	err = m.ReadAPIConfig()
	if err != nil {
		fmt.Println("reload consul setting failed", err)
	}
	for {
		select {
		case <-ticker:
			err = m.ReadIPSetConfig()
			if err != nil {
				fmt.Println("reload consul setting failed", err)
			}
			err = m.ReadAPIConfig()
			if err != nil {
				fmt.Println("reload consul setting failed", err)
			}
		case <-m.exitChan:
			return
		}
	}
}

func (m *DDoSAgent) ReadIPSetConfig() error {
	newConf, err := m.ReadConfigFromConsul(fmt.Sprintf("%s/%s", m.Setting["cluster"], "ipset"))
	if err != nil {
		return err
	}
	for k, _ := range newConf {
		if (m.IPSets[k] != nil) && (m.IPSets[k].HashSetName != newConf[k]) {
			m.IPSets[k].Stop()
			delete(m.IPSets, k)
			if len(newConf[k]) > 0 {
				ipset := &IPSet{}
				ipset.Topic = k
				ipset.HashSetName = newConf[k]
				ipset.HashName = fmt.Sprintf("%shash", newConf[k])
				ipset.Timeout = m.Setting["timeout"]
				ipset.MaxSize, _ = strconv.Atoi(m.Setting["max_size"])
				ipset.LookupdAddresses = strings.Split(m.Setting["lookupd_addresses"], ",")
				m.IPSets[k] = ipset
				go ipset.Run()
				m.RegisterService("ipset", k)
			}
		}
	}
	for k, _ := range m.IPSets {
		if m.IPSets[k].HashSetName != newConf[k] {
			if len(newConf[k]) == 0 {
				m.IPSets[k].Stop()
				delete(m.IPSets, k)
			}
		}
	}
	return nil
}

func (m *DDoSAgent) ReadAPIConfig() error {
	newConf, err := m.ReadConfigFromConsul(fmt.Sprintf("%s/%s", m.Setting["cluster"], "nginx"))
	if err != nil {
		return err
	}
	for k, _ := range newConf {
		if (m.APITasks[k] != nil) && (m.APITasks[k].EndPoint != newConf[k]) {
			m.APITasks[k].Stop()
			delete(m.APITasks, k)
			if len(newConf[k]) > 0 {
				apitask := &APITask{}
				apitask.Topic = k
				apitask.EndPoint = newConf[k]
				apitask.LookupdAddresses = strings.Split(m.Setting["lookupd_addresses"], ",")
				m.APITasks[k] = apitask
				go apitask.Run()
				m.RegisterService("nginx", k)
			}
		}
	}
	for k, _ := range m.APITasks {
		if m.APITasks[k].EndPoint != newConf[k] {
			if len(newConf[k]) == 0 {
				m.APITasks[k].Stop()
				delete(m.APITasks, k)
			}
		}
	}
	return nil
}

func (m *DDoSAgent) ReadConfigFromConsul(key string) (map[string]string, error) {
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
func (m *DDoSAgent) Status(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	c.Header("Access-Control-Allow-Methods", "GET")
	topic := c.Param("topic")
	serviceType := c.Param("serviceType")
	rst := fmt.Sprintf("%s_%s", serviceType, topic)
	if serviceType == "ipset" {
		if _, ok := m.IPSets[topic]; ok {
			c.String(http.StatusOK, rst)
			return
		}
	}
	if serviceType == "nginx" {
		if _, ok := m.APITasks[topic]; ok {
			c.String(http.StatusOK, rst)
			return
		}
	}
	c.String(http.StatusNotFound, rst)
}

func (m *DDoSAgent) showIPSet(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	c.Header("Access-Control-Allow-Methods", "GET")
	ipset := "/usr/sbin/ipset"
	if _, err := os.Stat(ipset); os.IsNotExist(err) {
		ipset = "/sbin/ipset"
	}
	if _, err := os.Stat(ipset); os.IsNotExist(err) {
		c.String(http.StatusInternalServerError, "ipset not found")
		return
	}
	cmd := exec.Command("/usr/bin/sudo", ipset, "list")
	if out, err := cmd.Output(); err != nil {
		c.String(http.StatusInternalServerError, "ipset list error")
	} else {
		c.String(http.StatusOK, string(out))
	}
}

func (m *DDoSAgent) showNginx(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=\"utf-8\"")
	c.Header("Access-Control-Allow-Methods", "GET")
	topic := c.Param("topic")
	apitask := m.APITasks[topic]
	if apitask == nil {
		c.String(http.StatusNotFound, "bad topic")
		return
	}
	var body []byte
	if strings.Contains(topic, "ip") {
		body = []byte("show_type=ip&show_list=all")
	} else {
		body = []byte("show_type=variable&show_list=all")
	}
	buf := bytes.NewBuffer(body)
	resp, err := apitask.client.Post(apitask.EndPoint, "application/x-www-form-urlencoded", buf)
	if err != nil {
		log.Println("connect timeout", err)
		c.String(http.StatusOK, "error to post"+apitask.EndPoint)
	} else {
		if resp.StatusCode != 200 {
			log.Printf("unsuccessfull return %s\n", resp.Status)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("read response", err)
		}
		resp.Body.Close()
		c.String(http.StatusOK, string(body))
	}
}

func (m *DDoSAgent) RegisterService(serviceType string, service string) error {
	rand.Seed(time.Now().UnixNano())
	sid := rand.Intn(65534)
	serviceID := fmt.Sprintf("%s-%s-%d", serviceType, service, sid)

	consulService := api.AgentServiceRegistration{
		ID:   serviceID,
		Name: service,
		Tags: []string{serviceType, service, time.Now().Format("Jan 02 15:04:05.000 MST")},
		Port: m.Port,
		Check: &api.AgentServiceCheck{
			Script:   fmt.Sprintf("curl --connect-timeout=5 http://%s:%d/status/%s/%s", m.HostName, m.Port, serviceType, service),
			Interval: "10s",
			Timeout:  "8s",
			TTL:      "",
			HTTP:     fmt.Sprintf("http://%s:%d/status/%s/%s", m.HostName, m.Port, serviceType, service),
			Status:   "passing",
		},
		Checks: api.AgentServiceChecks{},
	}
	err := m.agent.ServiceRegister(&consulService)
	if err != nil {
		return err
	}

	return err
}
