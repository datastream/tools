package main

import (
	"labix.org/v2/mgo"
	"log"
	"strings"
)

type Producer struct {
	session *mgo.Session
	collection *mgo.Collection
	done chan error
}

func NewProducer(mongouri, dbname, collection, user, password string) (m *Producer, err error) {
	m = new(Producer)
	m.session, err =  mgo.Dial(mongouri)
	if err != nil {
		return
	}
	db := m.session.DB(dbname)
	err = db.Login(user,password)
	if err != nil {
		return
	}
	m.collection = db.C(collection)
	return
}

func (this *Producer)handle(work *Work) {
	for {
		var err error
		body := <- work.message
		metrics := strings.Split(strings.TrimSpace(*body),"\n")
		for i := range metrics {
			err = this.collection.Insert(NewMetric(metrics[i]))
			if err != nil {
				log.Printf("mongodb insert failed")
				this.session.Close()
				break
			}
		}
		if err != nil {
			this.done <- err
			work.producer <- this
			work.message <- body
			break
		}
	}
}
