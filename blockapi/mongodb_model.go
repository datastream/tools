package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"strings"
)

type Mongo struct {
	session    *mgo.Session
	mongouri   string
	dbname     string
	collection string
	user       string
	password   string
	done       chan error
}

func NewMongo(mongouri, dbname, collection, user, password string) (m *Mongo, err error) {
	m = &Mongo{
		mongouri:   mongouri,
		dbname:     dbname,
		collection: collection,
		user:       user,
		password:   password,
	}
	m.session, err = mgo.DialWithTimeout(mongouri, 10)
	m.done = make(chan error)
	if err != nil {
		m.session = nil
		log.Println(err)
		return
	}
	err = m.session.DB(m.dbname).Login(user, password)
	if err != nil {
		m.session = nil
		log.Println(err)
		return
	}
	return
}
func (this *Mongo) handle(req WhiteListRequest) {
	session := this.session.Clone()
	defer session.Close()
	var err error
	var gateway [][]byte
	ip_list := strings.Split(req.ip, ",")
	for i := range ip_list {
		if len(ip_list[i]) < 7 {
			continue
		}
		var n int
		n, err = session.DB(this.dbname).C(this.collection).Find(bson.M{"ip": strings.TrimSpace(ip_list[i])}).Count()
		if err != nil {
			log.Printf("query error:%s\n", err)
			this.done <- nil
			break
		}
		if n > 0 {
			log.Println(ip_list[i], "is gateway")
			gateway = append(gateway, []byte(ip_list[i]))
		}
	}
	if len(gateway) > 1 {
		rq := &FirewallRequest{}
		rq.iprequest.RequestType = REQUEST_TYPE_DELTE.Enum()
		rq.iprequest.Ipaddresses = gateway
		for l := range req.hosts {
			go sendtohost(req.hosts[l], rq)
		}
	}
}
