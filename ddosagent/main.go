package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string]string

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	DA := &DDoSAgent{}
	DA.Setting = setting
	DA.Run()
	termchan := make(chan os.Signal, 1)
	signal.Notify(termchan, syscall.SIGINT, syscall.SIGTERM)
	<-termchan
	DA.Stop()
}
