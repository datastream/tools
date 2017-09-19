package main

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/nsqio/go-nsq"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type APITask struct {
	ClusterName      string
	Topic            string
	EndPoint         string
	LookupdAddresses []string
	consumer         *nsq.Consumer
	client           *http.Client
	nodeName         string
	UpdatedAt        int32
	agent            *DDoSAgent
}

func (t *APITask) Stop() {
	t.consumer.Stop()
}
func (t *APITask) Run() error {
	t.client = &http.Client{Timeout: time.Duration(5 * time.Second)}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	t.nodeName = hostname
	cfg := nsq.NewConfig()
	cfg.Set("user_agent", fmt.Sprintf("ddosAgent/%s", hostname))
	t.consumer, err = nsq.NewConsumer(t.Topic, hostname, cfg)
	t.consumer.AddHandler(t)
	err = t.consumer.ConnectToNSQLookupds(t.LookupdAddresses)
	return err
}
func (t *APITask) HandleMessage(m *nsq.Message) error {
	buf := bytes.NewBuffer(m.Body)
	resp, err := t.client.Post(t.EndPoint, "application/x-www-form-urlencoded", buf)
	if err != nil {
		return err
	}
	if resp.StatusCode == 400 {
		resp.Body.Close()
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("api error")
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	cur := atomic.LoadInt32(&t.UpdatedAt)
	if cur > 10 {
		if strings.Contains(string(m.Body), "variable") {
			go t.GC([]byte("show_type=variable&show_list=all"))
		} else {
			go t.GC([]byte("show_type=ip&show_list=all"))
		}
		atomic.StoreInt32(&t.UpdatedAt, 0)
	} else {
		atomic.AddInt32(&t.UpdatedAt, 1)
	}
	return nil
}
func (t *APITask) GC(args []byte) {
	buf := bytes.NewBuffer(args)
	resp, err := t.client.Post(t.EndPoint, "application/x-www-form-urlencoded", buf)
	if err != nil {
		return
	}
	if resp.StatusCode == 400 {
		return
	}
	if resp.StatusCode != 200 {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body.Close()
	t.cleanExpired(string(body))
	kv := &api.KVPair{Key: fmt.Sprintf("ddosagent/status/nginx/%s/%s", t.Topic, t.nodeName), Value: body}
	_, err = t.agent.client.KV().Put(kv, nil)
	if err != nil {
		return
	}
}

func (t *APITask) cleanExpired(body string) {
	lines := strings.Split(body, "\n")
	var lists []string
	dataType := "ip"
	for _, line := range lines {
		if !strings.Contains(line, "expire=expired") {
			continue
		}
		items := strings.Split(line, " ")
		if len(items) < 3 {
			continue
		}
		kv := strings.Split(items[1], "=")
		if len(kv) != 2 {
			continue
		}
		dataType = kv[0]
		if kv[0] == "ip" {
			ip := strings.Split(kv[1], "(")
			lists = append(lists, ip[0])
		} else {
			l := len(kv[1])
			if l > 3 {
				lists = append(lists, kv[1][1:l-2])
			}
		}
	}
	t.doRequest(fmt.Sprintf("free_type=%s&free_list=%s", dataType, strings.Join(lists, ",")))
}
func (t *APITask) doRequest(body string) {
	buf := bytes.NewBuffer([]byte(body))
	resp, err := t.client.Post(t.EndPoint, "application/x-www-form-urlencoded", buf)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		return
	}
	if resp.StatusCode != 200 {
		return
	}
	io.Copy(ioutil.Discard, resp.Body)
}
