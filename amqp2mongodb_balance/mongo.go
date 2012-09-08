package main

import (
	"labix.org/v2/mgo"
	"log"
	"regexp"
	"strings"
	"time"
)

type Mongo struct {
	session  *mgo.Session
	mongouri string
	dbname   string
	user     string
	password string
	done     chan error
}

func NewMongo(mongouri, dbname, user, password string) *Mongo {
	this := &Mongo{
		mongouri: mongouri,
		dbname:   dbname,
		user:     user,
		password: password,
	}
	return this
}

func (this *Mongo) connect_mongodb() {
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
func (this *Mongo) insert_record(message_chan chan *Message) {
	this.connect_mongodb()
	go this.handle_insert(message_chan)
	for {
		<-this.done
		this.session.Refresh()
		go this.handle_insert(message_chan)
	}
}

func (this *Mongo) handle_insert(message_chan chan *Message) {
	session := this.session.Copy()
	defer session.Close()
	for {
		var err error
		msg := <-message_chan
		metrics := strings.Split(strings.TrimSpace(msg.content), "\n")
		for i := range metrics {
			record := NewMetric(metrics[i])
			if record != nil {
				if rst, _ := regexp.MatchString("sd[a-z]{1,2}[0-9]{1,2}", record.Nm); rst && record.App == "disk" {
					continue
				}
				if rst, _ := regexp.MatchString("(eth|br|bond)[0-9]{1,2}", record.Nm); !rst && record.App == "interface" {
					continue
				}
				err = session.DB(this.dbname).C(record.App).Insert(record.Record)
				splitname := strings.Split(metrics[i], " ")
				host := &Host{
					Host:   record.Hs,
					Metric: splitname[0],
					Ttl:    -1,
				}
				err = session.DB(this.dbname).C("host_metric").Insert(host)

				if err != nil {
					if rst, _ := regexp.MatchString("dup", err.Error()); rst {
						err = nil
					} else {
						log.Println("mongodb insert failed", err)
						this.done <- err
						break
					}
				}
			} else {
				log.Println("metrics error:", msg.content)
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