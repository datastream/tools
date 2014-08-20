package main

import (
	"flag"
	"fmt"
	"github.com/bitly/go-nsq"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	bind   = flag.String("port", "0.0.0.0:1234", "http port")
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string]string
var ipSet *IPSet

// IPSet store info about ipset hash/list name
type IPSet struct {
	HashSetName string
	HashName    string
	HashList    []string
	maxsize     int
	timeout     string
}

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	ipSet = &IPSet{
		HashName:    setting["hashname"],
		HashSetName: setting["hashsetname"],
		maxsize:     8,
		timeout:     setting["timeout"],
	}
	ipSet.setup()
	ddosChannel, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	topics := strings.Split(setting["topics"], ",")
	cfg := nsq.NewConfig()
	hostname, err := os.Hostname()
	cfg.Set("user_agent", fmt.Sprintf("ipsetclient/%s", hostname))
	var ipSetTasks []*nsq.Consumer
	for _, topic := range topics {
		consumer, err := nsq.NewConsumer(topic, ddosChannel, cfg)
		consumer.AddHandler(ipSet)
		lookupdlist := strings.Split(setting["lookupdaddresses"], ",")
		err = consumer.ConnectToNSQLookupds(lookupdlist)
		if err != nil {
			log.Fatal(err)
		}
		ipSetTasks = append(ipSetTasks, consumer)
	}
	for _, consumer := range ipSetTasks {
		defer consumer.Stop()
	}
	http.HandleFunc("/", showIP)
	err = http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func showIP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=\"utf-8\"")
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "list")
	if out, err := cmd.Output(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}
}
