package main

import (
	"labix.org/v2/mgo"
	"log"
	"strings"
	"time"
)

type Producer struct {
	session  *mgo.Session
	mongouri string
	dbname   string
	user     string
	password string
	done     chan error
}

func NewProducer(mongouri, dbname, user, password string) *Producer {
	this := &Producer{
		mongouri: mongouri,
		dbname:   dbname,
		user:     user,
		password: password,
	}
	return this
}

func (this *Producer) connect_mongodb() {
	var err error
	for {
		this.session, err = mgo.Dial(this.mongouri)
		if err != nil {
			time.Sleep(time.Second * 2)
			continue
		}
		if len(this.user) > 0 {
			err = this.session.DB(this.dbname).Login(this.user, this.password)
			if err != nil {
				time.Sleep(time.Second * 2)
				continue
			}
		}
		break
	}
}
func (this *Producer) insert_record(message_chan chan *Message) {
	this.connect_mongodb()
	for {
		select {
		case <-this.done:
			{
				this.session.Refresh()
			}
		case msg := <-message_chan:
			{
				go func() {
					if this.handle(msg) {
						message_chan <- msg

					}
				}()
			}
		}
	}
}

func (this *Producer) handle(msg *Message) bool {
	session := this.session.Clone()
	defer session.Close()
	var err error
	metrics := strings.Split(strings.TrimSpace(msg.content), "\n")
	for i := range metrics {
		record := NewMetric(metrics[i])
		if record != nil {
			err = session.DB(this.dbname).C("monitor_data").Insert(record)
			splitname := strings.Split(metrics[i], " ")
			host := &Host{
				Host:   record.Hostname,
				Metric: splitname[0],
				Ttl:    -1,
			}
			err = session.DB(this.dbname).C("host_metric").Insert(host)

			if err != nil {
				if err.(*mgo.LastError).Code == 11000 {
					err = nil
				} else {
					log.Println("mongodb insert failed", err)
					this.done <- err
					break
				}
			}
		} else {
			log.Println("metrics error", metrics[i])
		}
	}
	if err != nil {
		return false
	}
	msg.done <- 1
	return true
}
