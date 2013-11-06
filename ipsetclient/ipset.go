package main

import (
	"github.com/bitly/go-nsq"
	"log"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (s *IPSet) setup() {
	s.setupHashSet()
	s.setupIPHash()
}

func (s *IPSet) setupHashSet() error {
	_, err := exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "create", s.HashSetName, "list:set", "-exist").Output()
	if err != nil {
		log.Fatal("ipset create setlist failed:", err)
	}
	_, err = exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "flush", s.HashSetName).Output()
	return err
}

func (s *IPSet) setupIPHash() {
	s.HashList = s.HashList[:0]
	for index := 0; index < s.maxsize; index++ {
		name := s.HashName + strconv.Itoa(index)
		s.HashList = append(s.HashList, name)
		_, err := exec.Command("/usr/bin/sudo",
			"/usr/sbin/ipset", "create", name, "hash:ip", "timeout", s.timeout, "-exist").Output()
		if err != nil {
			log.Fatal("ipset create iphash ", name, " failed:", err)
		}
		s.addHashSet(name)
	}
}

func (s *IPSet) addHashSet(name string) {
	_, err := exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "add", s.HashSetName, name, "-exist").Output()
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
		timeout = s.timeout
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
	return nil
}

func (s *IPSet) delIP(ipaddresses []string) {
	for _, ip := range ipaddresses {
		for _, hashname := range s.HashList {
			exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "del", hashname, ip).Output()
		}
	}
}
func (s *IPSet) clearIP() {
	for _, h := range s.HashList {
		exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "flush", h).Output()
	}
}
func (s *IPSet) updateIP(ipaddresses []string, timeout string) {
	for _, ip := range ipaddresses {
		if len(ip) < 7 {
			return
		}
		for index := 0; index < s.maxsize; index++ {
			hashname := s.HashList[index]
			out, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "add", hashname, ip, "timeout", timeout, "-exist").CombinedOutput()
			if err != nil {
				log.Println("add ip failed", out, err)
				reg, e := regexp.Compile("is full")
				if e == nil && reg.MatchString(string(out)) {
					continue
				}
			}
			break
		}
	}
}
