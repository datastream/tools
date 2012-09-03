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
	for {
		select {
		case req := <-check_chan:
			{
				go m.handle(req)
			}

		case <-m.done:
			{
				m.session.Refresh()
			}
		}
	}
}
