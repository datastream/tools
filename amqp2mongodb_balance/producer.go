package main

import (
	"labix.org/v2/mgo"
	"log"
	"regexp"
	"strings"
	"time"
)

type Producer struct {
	session  *mgo.Session
	db       *mgo.Database
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
	for {
		session, err := mgo.Dial(this.mongouri)
		if err != nil {
			time.Sleep(time.Second * 2)
			continue
		}
		this.db = session.DB(this.dbname)
		if len(this.user) > 0 {
			err = this.db.Login(this.user, this.password)
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
	go this.handle(message_chan)
	for {
		<-this.done
		this.connect_mongodb()
		go this.handle(message_chan)
	}
}

func (this *Producer) handle(message_chan chan *Message) {
	for {
		var err error
		msg := <-message_chan
		metrics := strings.Split(strings.TrimSpace(msg.content), "\n")
		for i := range metrics {
			record := NewMetric(metrics[i])
			if record != nil {
				err = this.db.C("monitor_data").Insert(record)
				splitname := strings.Split(metrics[i], " ")
				host := &Host{
					Host:   record.Hostname,
					Metric: splitname[0],
					Ttl:    -1,
				}
				err = this.db.C("host_metric").Insert(host)

				if err != nil {
					if ok, _ := regexp.MatchString("duplicate key", err.Error()); ok {
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
			go func() {
				message_chan <- msg
			}()
			break
		}
		msg.done <- 1
	}
}
