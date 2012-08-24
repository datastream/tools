package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"log"
)

type Mongo struct {
	session    *mgo.Session
	collection *mgo.Collection
	done       chan error
}

func NewMongo(mongouri, dbname, collection, user, password string) (m *Mongo, err error) {
	m = new(Mongo)
	m.session, err = mgo.DialWithTimeout(mongouri, 10)
	m.done = make(chan error)
	log.Println("New session")
	if err != nil {
		m.session = nil
		return
	}
	db := m.session.DB(dbname)
	err = db.Login(user, password)
	if err != nil {
		m.session = nil
		return
	}
	m.collection = db.C(collection)
	return
}
func (this *Mongo) handle(check_chan chan ipitem) {
	for {
		req := <- check_chan
		var err error
		var gateway string
		ip_list := strings.Split(req.ip, ",")
		for i := range ip_list {
			if len(ip_list[i]) < 7 {
				continue
			}
			var n int
			n, err = this.collection.Find(bson.M{"ip":strings.TrimSpace(ip_list[i])}).Count()
			if err != nil {
				log.Printf("query error:%s\n", err)
				this.done <- nil
				break
			}
			if n > 0 {
				log.Println(ip_list[i],"is gateway")
				gateway += ip_list[i] + ","
			}
		}
		for l := range req.hosts {
			if len(gateway) > 1 {
				erro := make(chan error)
				go handle(strings.TrimSpace(req.hosts[l]), "del",
					gateway[:len(gateway)-1], "", erro)
				<- erro

			}
		}
	}
}
