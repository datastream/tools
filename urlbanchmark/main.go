package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	confFile = flag.String("c", "urlbanchmark.json", "url banchmark config file")
)

func main() {
	flag.Parse()
	c, err := ReadConfig(*confFile)
	if err != nil {
		log.Fatal("parse config file error: ", err)
	}
	p := &TaskPool{
		Setting:     c,
		exitChannel: make(chan int),
		msgChannel:  make(chan []byte),
	}
	p.Run()
	termchan := make(chan os.Signal, 1)
	signal.Notify(termchan, syscall.SIGINT, syscall.SIGTERM)
	<-termchan
	p.Stop()
}
