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
	if resp.StatusCode != 200 {
		return fmt.Errorf("api error")
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	if strings.Contains(string(m.Body), "variable") {
		buf = bytes.NewBuffer([]byte("show_type=variable&show_list=all"))
	} else {
		buf = bytes.NewBuffer([]byte("show_type=ip&show_list=all"))
	}
	resp, err = t.client.Post(t.EndPoint, "application/x-www-form-urlencoded", buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("api error")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body")
	}
	kv := &api.KVPair{Key: fmt.Sprintf("%s/status/nginx/%s/%s", t.ClusterName, t.Topic, t.nodeName), Value: body}
	_, err = t.agent.client.KV().Put(kv, nil)
	if err != nil {
		return fmt.Errorf("write consul failed")
	}
	return nil
}
