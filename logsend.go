package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"flag"
	"io"
)

var (
	filename          = flag.String("file", "/var/log/apache/access.log", "filenema")
)
func init() {
	flag.Parse()
}
var done chan int

func main() {
	logchan := make(chan *[]byte)
	done = make(chan int)
	go run_server(logchan)
	go read_log(logchan)
	<-done
}
func read_log(logchan chan *[]byte) {
	cmd := exec.Command("/bin/cat", *filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Read Pipe error", err)
	}
	f := bufio.NewReaderSize(stdout, 10240)
	line := make([]byte,1024)
	err = cmd.Start()
	if err != nil {
		fmt.Println("run error", err)
	}
	for {
		n, err := f.Read(line)
		if err == io.EOF {
			done <- 1
			break
		}
		line = line[0:n]
		logchan <- &line
	}
}
func send_log(fd net.Conn, logchan chan *[]byte) {
	defer fd.Close()
	for {
		msg := <-logchan
		_, err := fd.Write(*msg)
		if err != nil {
			fmt.Printf("TCP connect write error")
			logchan <- msg
			break
		}
	}
	fmt.Printf("TCP closed!\n")
}
func run_server(log chan *[]byte) {
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
