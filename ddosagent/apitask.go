package main

import (
	"bytes"
	"fmt"
	"github.com/nsqio/go-nsq"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type APITask struct {
	Topic            string
	EndPoint         string
	LookupdAddresses []string
	consumer         *nsq.Consumer
	client           *http.Client
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
	return nil
}
