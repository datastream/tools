package main

import (
	"time"
)

func connect_db(mongouri, dbname, collection, user, password string) *Mongo {
	for {
		m, err := NewMongo(mongouri, dbname, collection, user, password)
		if err != nil {
			time.Sleep(time.Second * 2)
		} else {
			return m
		}
	}
	return nil
}

func check_whitelist(check_chan chan WhiteListRequest) {
	m := connect_db(*mongouri, *dbname, *collection, *user, *password)
	go m.handle(check_chan)
	for {
		<-m.done
		m.session.Close()
		m = connect_db(*mongouri, *dbname, *collection, *user, *password)
		go m.handle(check_chan)
	}
}
