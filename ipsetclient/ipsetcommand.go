package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func check_iphash() {
	output, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", *blockset).Output()
	if err == nil {
		buf := bytes.NewBuffer(output)
		i := false
		for {
			line, _ := buf.ReadString('\n')
			if line[:len(line)-1] == "Members:" {
				i = true
				continue
			}
			if line[:len(line)-1] == "Bindings:" {
				break
			}
			if i {
				hashlist = append(hashlist, line[:len(line)-1])
			}
		}
	}
}
func create_set(setname string) {
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", setname).Output()
	if err == nil {
		log.Println("setlist ", *blockset, " exist!")
		return
	}
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", setname, "setlist")
	if err = cmd.Run(); err != nil {
		log.Println("ipset create setlist failed:", err)
	}
}

func create_hash(name string) {
	check_iphash()
	full := false
	if len(hashlist) >= 8 {
		if index > 7 {
			index = 0
		}
		hashname = hashlist[index]
		name = hashname
		full = true
		log.Println("setlist full, reuse ", hashname)
	}
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-L", name).Output()
	if err == nil {
		log.Println("iphash ", name, " exist!")
		if full {
			_, _ = exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-F", name).Output()
		}
		return
	}
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-N", name, "iphash")
	hashlist = append(hashlist, name)
	if err := cmd.Run(); err != nil {
		log.Println("ipset create iphash ", name, " failed:", err)
	}
}

func add_hashlist(hash string) {
	_, err := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", *blockset, hash).Output()
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-A", *blockset, hash)
	if err = cmd.Run(); err != nil {
		log.Println("ipset add ", hash, " to ", *blockset, " setlist failed:", err)
	}
}

func expire_ip(expire_chan chan *ipset, sleep_chan chan int32) {
	for {
		select {
		case item := <-expire_chan:
			{
				cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ipset", "-D", item.set, item.ip)
				err := cmd.Run()
				if err != nil {
					log.Println(item.set, ": ", item.ip, " auto delete error", err)
				} else {
					log.Println("auto expire:", item.set, item.ip)
				}
			}
		case i := <-sleep_chan:
			{
				time.Sleep(time.Second * time.Duration(i))
				log.Println("stop auto expire")
			}
		}
	}
}

func read_speed(speed_chan chan uint64, sleep_chan chan int32) {
	fd, err := os.Open("/sys/class/net/eth0/statistics/rx_bytes")
	if err != nil {
		log.Println("fail to read /sys/class/net/eth0/statistics/rx_bytes")
		return
	}
	reader := bufio.NewReader(fd)
	var line string
	for {
		fd.Seek(0, 0)
		line, _ = reader.ReadString('\n')
		stat1, _ := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
		time.Sleep(time.Second * 5)
		fd.Seek(0, 0)
		line, _ = reader.ReadString('\n')
		stat2, _ := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
		speed := (stat2 - stat1) / 5 / 1024 / 1024
		speed_chan <- speed
		if int(speed) > *hardlimit {
			sleep_chan <- int32(speed) * 6
		}
		time.Sleep(time.Second * 5)
	}
}
