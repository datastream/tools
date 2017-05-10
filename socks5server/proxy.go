package main

import (
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

var seed int64
var ipList []net.IP

func rfc1918private(ip net.IP) bool {
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("failed to parse hardcoded rfc1918 cidr: " + err.Error())
		}
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}

func getAllIPv4PublicIP() ([]net.IP, error) {
	var IPAddresses []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return IPAddresses, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Println(err)
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			private := rfc1918private(ip)
			if !(private || ip.IsLoopback()) && (ip.To4() != nil) {
				IPAddresses = append(IPAddresses, ip)
			}
		}
	}
	return IPAddresses, nil
}
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ipList, _ = getAllIPv4PublicIP()
	seed = 42
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Panic(err)
	}

	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}

	var b [1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}
	var wg sync.WaitGroup
	if b[0] == 0x05 {
		client.Write([]byte{0x05, 0x00})
		n, err = client.Read(b[:])
		var host, port string
		switch b[3] {
		case 0x01: //IP V4
			host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		case 0x03: // domain
			host = string(b[5 : n-2])
		case 0x04: //IP V6
			host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}.String()
		}
		port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))
		var server net.Conn
		for {
			rand.Seed(time.Now().Unix())
			localeAddr := &net.TCPAddr{
				IP:   ipList[rand.Intn(len(ipList))],
				Port: rand.Intn(64510) + 1024,
			}
			seed = int64(rand.Intn(64510))
			log.Println(localeAddr)
			remouteAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(remouteAddr)
			server, err = net.DialTCP("tcp", localeAddr, remouteAddr)
			if err != nil {
				log.Println(err)
				continue
			}
			break
		}
		client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		wg.Add(1)
		go func() {
			io.Copy(client, server)
			client.Close()
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			io.Copy(server, client)
			server.Close()
			wg.Done()
		}()
		wg.Wait()
	}
	log.Println("close connect")
}
