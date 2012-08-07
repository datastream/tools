package main

import (
	"labix.org/v2/mgo"
	"log"
	"strings"
)

type Producer struct {
	session    *mgo.Session
	collection *mgo.Collection
	done       chan error
}

func NewProducer(mongouri, dbname, collection, user, password string) (m *Producer, err error) {
	m = new(Producer)
	m.session, err = mgo.Dial(mongouri)
	if err != nil {
		return
	}
	db := m.session.DB(dbname)
	err = db.Login(user, password)
	if err != nil {
		return
	}
	m.collection = db.C(collection)
	return
}

func (this *Producer) handle(work *Work) {
	for {
		var err error
		msg := <-work.message
		metrics := strings.Split(strings.TrimSpace(msg.content), "\n")
		for i := range metrics {
			err = this.collection.Insert(NewMetric(metrics[i]))
			if err != nil {
				log.Printf("mongodb insert failed")
				this.session.Close()
				this.session = nil
				work.producer <- this
				this.done <- err
				break
			}
		}
		if err != nil {
			go func() {
				work.message <- msg
			}()
			break
		}
		msg.done <- 1
	}
}
