package main

import (
	"encoding/json"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/nsqio/go-nsq"
	"log"
	"os"
	"time"
)

type Builder struct {
	*Setting
	consumer    *nsq.Consumer
	msgChannel  chan *Message
	exitChannel chan int
}

func (m *Builder) Run() error {
	var err error
	cfg := nsq.NewConfig()
	hostname, err := os.Hostname()
	cfg.Set("user_agent", fmt.Sprintf("metric_processor/%s", hostname))
	cfg.Set("snappy", true)
	cfg.Set("max_in_flight", m.MaxInFlight)
	m.consumer, err = nsq.NewConsumer(m.Topic, m.Channel, cfg)
	if err != nil {
		log.Println(m.Topic, err)
		return err
	}
	hystrix.ConfigureCommand("InsetInfluxdb", hystrix.CommandConfig{
		Timeout:               1000,
		MaxConcurrentRequests: 1000,
		ErrorPercentThreshold: 25,
	})
	go m.writeLoop()
	m.consumer.AddConcurrentHandlers(m, m.MaxInFlight)
	err = m.consumer.ConnectToNSQLookupds(m.LookupdAddresses)
	if err != nil {
		return err
	}
	return err
}

type Message struct {
	Body         []byte
	ErrorChannel chan error
}

func (m *Builder) HandleMessage(msg *nsq.Message) error {
	message := &Message{
		Body:         msg.Body,
		ErrorChannel: make(chan error),
	}
	m.msgChannel <- message
	err := <-message.ErrorChannel
	if err != nil {
		log.Println(err)
	}
	return err
}
func (m *Builder) writeLoop() {
	db, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:      m.InfluxdbAddress,
		Username:  m.InfluxdbUser,
		Password:  m.InfluxdbPassword,
		UserAgent: fmt.Sprintf("smartinfo-%s", VERSION),
	})
	if err != nil {
		log.Println("NewHTTPClient error:", err)
	}
	defer db.Close()
	q := client.NewQuery(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", m.InfluxdbDatabase), "", "s")
	if response, err := db.Query(q); err != nil || response.Error() != nil {
		log.Fatal("create influxdb database failed:", response.Results)
	}
	for {
		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  m.InfluxdbDatabase,
			Precision: "s",
		})
		select {
		case msg := <-m.msgChannel:
			dataset := make(map[string]interface{})
			var err error
			err = json.Unmarshal(msg.Body, &dataset)
			if err != nil {
				log.Println("wrong data struct:", string(msg.Body))
				msg.ErrorChannel <- nil
				continue
			}
			hostname := dataset["Hostname"].(string)
			t := dataset["Checktime"].(string)
			timestamp := time.Now()
			for k, v := range dataset {
				tags := make(map[string]string)
				fields := make(map[string]interface{})
				if diskinfo, ok := v.(map[string]interface{}); ok {
					tags["Hostname"] = hostname
					tags["Postion"] = k
					tags["Checktime"] = t
					for dk, dv := range diskinfo {
						if value, ok := dv.(string); ok {
							tags[dk] = value
						} else {
							fields[dk] = dv
						}
					}
				}
				var pt *client.Point
				pt, err = client.NewPoint("diskstat", tags, fields, timestamp)
				if err == nil {
					bp.AddPoint(pt)
				}
			}
			if err == nil {
				resultChan := make(chan int, 1)
				errChan := hystrix.Go("InsetInfluxdb", func() error {

					err = db.Write(bp)
					if err != nil {
						return err
					}
					resultChan <- 1
					return nil
				}, nil)
				select {
				case <-resultChan:
				case err = <-errChan:
					log.Println("InsetInfluxdb Error", err)
				}
			}
			msg.ErrorChannel <- err
		case <-m.exitChannel:
			return
		}
	}
}

func (m *Builder) Stop() {
	m.consumer.Stop()
	close(m.exitChannel)
}
