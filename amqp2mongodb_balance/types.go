package main

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

type Record struct {
	Rt string
	Nm string
	Cl string
	Hs string
	V  float64
	Ts int64
}

type Metric struct {
	Record
	App string
}

type Host struct {
	Host   string
	Metric string
	Ttl    int
}

func NewMetric(s string) *Metric {
	splitstring := strings.Split(s, " ")
	splitname := strings.Split(splitstring[0], ".")
	if len(splitname) < 3 {
		log.Println("metrics not match", s)
		return nil
	}
	var App string
	var Retention string
	var Name string
	var Hostname string
	var Colo string
	var Value float64
	var Timestamp int64
	Hostname = splitname[len(splitname)-1]
	Colo = splitname[len(splitname)-2]
	var p int
	if rst, _ := regexp.MatchString("(1sec|10sec|1min|5min)", splitname[0]); rst {
		Retention = splitname[0]
		App = splitname[1]
		p = 2
	} else {
		App = splitname[0]
		p = 1
	}

	for i := p; i < len(splitname)-2; i++ {
		if len(Name) > 0 {
			Name += "."
		}
		Name += splitname[i]
	}
	if len(splitstring) == 3 {
		Value, _ = strconv.ParseFloat(splitstring[1], 64)
		Timestamp, _ = strconv.ParseInt(splitstring[2], 10, 64)
		this := &Metric{
			App: App,
			Record: Record{
				Rt: Retention,
				Nm: Name,
				Cl: Colo,
				Hs: Hostname,
				V:  Value,
				Ts: Timestamp,
			},
		}
		return this
	} else if len(splitstring) != 1 {
		log.Println("metric not recognized: ", s)
	}
	return nil
}
