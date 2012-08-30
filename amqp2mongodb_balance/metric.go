package main

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

type Metric struct {
	Retention string
	App       string
	Name      string
	Colo      string
	Hostname  string
	Value     float64
	Timestamp int64
}

func NewMetric(s string) *Metric {
	this := new(Metric)
	splitstring := strings.Split(s, " ")
	splitname := strings.Split(splitstring[0], ".")
	if len(splitname) < 3 {
		log.Println("metrics not match", s)
		return nil
	}
	this.Hostname = splitname[len(splitname)-1]
	this.Colo = splitname[len(splitname)-2]
	var p int
	if rst, _ := regexp.MatchString("(1sec|10sec|1min|5min)", splitname[0]); rst {
		this.Retention = splitname[0]
		this.App = splitname[1]
		p = 2
	} else {
		this.App = splitname[0]
		p = 1
	}

	for i := p; i < len(splitname)-2; i++ {
		if len(this.Name) > 0 {
			this.Name += "."
		}
		this.Name += splitname[i]
	}
	if len(splitstring) == 3 {
		this.Value, _ = strconv.ParseFloat(splitstring[1], 64)
		this.Timestamp, _ = strconv.ParseInt(splitstring[2], 10, 64)
		return this
	} else if len(splitstring) != 1 {
		log.Println("metric not recognized: ", s)
	}
	return nil
}
