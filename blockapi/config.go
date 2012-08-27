package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
)

func read_config(name string) []string {
	fd, err := os.Open("./" + name)
	var lines []string
	if err != nil {
		log.Println("server list file open Error:", name)
	} else {
		f := bufio.NewReader(fd)
		for {
			line, err := f.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("file readline error")
				break
			}
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
