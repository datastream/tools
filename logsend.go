package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
	"flag"
)

var (
	filename          = flag.String("file", "/var/log/apache/access.log", "filenema")
)
func init() {
	flag.Parse()
}

func main() {
	logchan := make(chan []byte)
	done := make(chan int)
	go run_server(logchan)
	go read_log(logchan)
	<-done
}
func read_log(logchan chan []byte) {
	file, err := os.Open(*filename)
	if err != nil {
		fmt.Printf("Read Pipe error")
	}
	f := bufio.NewReaderSize(file, 10240)
	line := make([]byte,1024)
	for {
		n, err := f.Read(line)
		if err != nil {
			time.Sleep(1 * 1e7)
		}
		logchan <- line[0:n]
	}
}
func send_log(fd net.Conn, logchan chan []byte) {
	defer fd.Close()
	for {
		msg := <-logchan
		_, err := fd.Write(msg)
		if err != nil {
			fmt.Printf("TCP connect write error")
			logchan <- msg
			break
		}
	}
	fmt.Printf("TCP closed!\n")
}
func run_server(log chan []byte) {
	lp, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		fmt.Printf("Bind 1234 failed")
		return
	}
	defer lp.Close()
	for {
		fd, error := lp.Accept()
		if error != nil {
			fmt.Printf("accpet error %s", error)
		}
		go send_log(fd, log)
	}
}
