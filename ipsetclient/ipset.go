package main

import (
	"bytes"
	"github.com/bitly/nsq/nsq"
	"log"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (this *IPSet) setup() {
	this.setup_hashset()
	this.setup_iphash()
}

func (this *IPSet) setup_hashset() error {
	_, err := exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "create", this.HashSetName, "list:set", "-exist").Output()
	if err != nil {
		log.Fatal("ipset create setlist failed:", err)
	}
	_, err = exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "flush", this.HashSetName).Output()
	return err
}

func (this *IPSet) setup_iphash() {
	this.HashList = this.HashList[:0]
	for this.index = 0; this.index < this.maxsize; this.index++ {
		name := this.HashName + strconv.Itoa(this.index)
		this.HashList = append(this.HashList, name)
		_, err := exec.Command("/usr/bin/sudo",
			"/usr/sbin/ipset", "create", name, "hash:ip", "timeout", this.timeout, "-exist").Output()
		if err != nil {
			log.Fatal("ipset create iphash ", name, " failed:", err)
		}
		this.add_hashset(name)
	}
	this.index = 0
}

func (this *IPSet) add_hashset(name string) {
	_, err := exec.Command("/usr/bin/sudo",
		"/usr/sbin/ipset", "add", this.HashSetName, name, "-exist").Output()
	if err != nil {
		log.Fatal("ipset add ", name, " to ", this.HashSetName, " setlist failed:", err)
	}
}

func (this *IPSet) HandleMessage(m *nsq.Message) error {
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
		timeout = this.timeout
	}
	switch action {
	case "add":
		go this.update_ip(ipaddresses, timeout)
		log.Println("add", ipaddresses, timeout)
	case "del":
		go this.del_ip(ipaddresses)
		log.Println("del", ipaddresses)
	case "clear":
		go this.clear_ip()
	case "update":
		go this.update_ip(ipaddresses, timeout)
		log.Println("update", ipaddresses)
	default:
		log.Println("ignore action:", action)
	}
	return nil
}

func (this *IPSet) del_ip(ipaddresses []string) {
	for _, ip := range ipaddresses {
		for _, hashname := range this.HashList {
			exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "del", hashname, ip).Output()
		}
	}
}
func (this *IPSet) clear_ip() {
	for _, h := range this.HashList {
		exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "flush", h).Output()
	}
}
func (this *IPSet) update_ip(ipaddresses []string, timeout string) {
	for _, ip := range ipaddresses {
		if len(ip) < 7 {
			return
		}
		hashname := this.HashList[this.index]
		c := exec.Command("/usr/bin/sudo",
			"/usr/sbin/ipset",
			"add", hashname, ip,
			"timeout", timeout, "-exist")
		err := c.Run()
		if err != nil {
			var output bytes.Buffer
			c.Stderr = &output
			reg, e := regexp.Compile("is full")
			if e == nil && reg.MatchString(output.String()) {
				this.Lock()
				if this.index < this.maxsize {
					this.index++
				} else {
					this.index = 0
				}
				this.Unlock()
				this.update_ip([]string{ip}, timeout)
			}
			if e != nil {
				log.Fatal("add ip failed", e)
			}
		}
	}
}
