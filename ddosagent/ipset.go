package main

import (
	"fmt"
	"github.com/nsqio/go-nsq"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type IPSet struct {
	ClusterName      string
	Topic            string
	HashSetName      string
	HashName         string
	HashList         []string
	MaxSize          int
	Timeout          string
	ipset            string
	LookupdAddresses []string
	consumer         *nsq.Consumer
	nodeName         string
	agent            *DDoSAgent
}

func (s *IPSet) Run() error {
	err := s.setup()
	if err != nil {
		return err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	cfg := nsq.NewConfig()
	cfg.Set("user_agent", fmt.Sprintf("ddosAgent/%s", hostname))
	s.nodeName = hostname
	s.consumer, err = nsq.NewConsumer(s.Topic, hostname, cfg)
	s.consumer.AddHandler(s)
	err = s.consumer.ConnectToNSQLookupds(s.LookupdAddresses)
	return err
}

func (s *IPSet) Stop() {
	s.consumer.Stop()
}

func (s *IPSet) setup() error {
	s.ipset = "/usr/sbin/ipset"
	if _, err := os.Stat(s.ipset); os.IsNotExist(err) {
		s.ipset = "/sbin/ipset"
	}
	if _, err := os.Stat(s.ipset); os.IsNotExist(err) {
		return err
	}
	s.setupHashSet()
	s.setupIPHash()
	return nil
}

func (s *IPSet) setupHashSet() error {
	_, err := exec.Command("/usr/bin/sudo",
		s.ipset, "create", s.HashSetName, "list:set", "-exist").Output()
	if err != nil {
		log.Fatal("ipset create setlist failed:", err)
	}
	_, err = exec.Command("/usr/bin/sudo",
		s.ipset, "flush", s.HashSetName).Output()
	return err
}

func (s *IPSet) setupIPHash() {
	s.HashList = s.HashList[:0]
	for index := 0; index < s.MaxSize; index++ {
		name := s.HashName + strconv.Itoa(index)
		s.HashList = append(s.HashList, name)
		_, err := exec.Command("/usr/bin/sudo",
			s.ipset, "create", name, "hash:ip", "timeout", s.Timeout, "-exist").Output()
		if err != nil {
			log.Fatal("ipset create iphash ", name, " failed:", err)
		}
		s.addHashSet(name)
	}
}

func (s *IPSet) addHashSet(name string) {
	_, err := exec.Command("/usr/bin/sudo",
		s.ipset, "add", s.HashSetName, name, "-exist").Output()
	if err != nil {
		log.Fatal("ipset add ", name, " to ", s.HashSetName, " setlist failed:", err)
	}
}

// HandleMessage for ipset nsq
func (s *IPSet) HandleMessage(m *nsq.Message) error {
	req, e := url.ParseQuery(string(m.Body))
	if e != nil {
		log.Println("bad req", string(m.Body), e)
		return nil
	}
	var action string
	if len(req["action_type"]) > 0 {
		action = req["action_type"][0]
	}
	var ipaddresses []string
	if len(req["ip"]) > 0 {
		ips := req["ip"]
		for _, v := range ips {
			items := strings.Split(v, ",")
			ipaddresses = append(ipaddresses, items...)
		}
	}
	var timeout string
	if len(req["timeout"]) > 0 {
		timeout = req["timeout"][0]
	} else {
		timeout = s.Timeout
	}
	switch action {
	case "add":
		go s.updateIP(ipaddresses, timeout)
		log.Println("add", ipaddresses, timeout)
	case "del":
		go s.delIP(ipaddresses)
		log.Println("del", ipaddresses)
	case "clear":
		go s.clearIP()
	case "update":
		go s.updateIP(ipaddresses, timeout)
		log.Println("update", ipaddresses)
	default:
		log.Println("ignore action:", action)
	}
	ipset := "/usr/sbin/ipset"
	if _, err := os.Stat(ipset); os.IsNotExist(err) {
		ipset = "/sbin/ipset"
	}
	cmd := exec.Command("/usr/bin/sudo", ipset, "list")
	if out, err := cmd.Output(); err == nil {
		err = s.agent.redisClient.Set(fmt.Sprintf("ddosagent/status/ipset/%s/%s", s.Topic, s.nodeName), out, 0).Err()
		if err != nil {
			return fmt.Errorf("write consul failed")
		}
	}
	return nil
}

func (s *IPSet) delIP(ipaddresses []string) {
	for _, ip := range ipaddresses {
		for _, hashname := range s.HashList {
			exec.Command("/usr/bin/sudo", s.ipset, "del", hashname, ip).Output()
		}
	}
}
func (s *IPSet) clearIP() {
	for _, h := range s.HashList {
		exec.Command("/usr/bin/sudo", s.ipset, "flush", h).Output()
	}
}
func (s *IPSet) updateIP(ipaddresses []string, timeout string) {
	for _, ip := range ipaddresses {
		if len(ip) < 7 {
			return
		}
		for index := 0; index < s.MaxSize; index++ {
			hashname := s.HashList[index]
			out, err := exec.Command("/usr/bin/sudo", s.ipset, "add", hashname, ip, "timeout", timeout, "-exist").CombinedOutput()
			if err != nil {
				log.Println("add ip failed", out, err)
				reg, e := regexp.Compile("is full")
				if e == nil && reg.MatchString(string(out)) {
					continue
				}
			}
			out, err = exec.Command("/usr/bin/sudo", s.ipset, "test", hashname, ip).CombinedOutput()
			if err == nil {
				reg, e := regexp.Compile("NOT")
				if e == nil && reg.MatchString(string(out)) {
					continue
				}
			} else {
				log.Println("test ip failed", out, err)
			}
			break
		}
	}
}
