package main

import (
	"fmt"
	"os/exec"
	"time"
)

func main() {
	ticker := time.Tick(time.Minute * 30)
	ticker2 := time.Tick(time.Hour * 24)
	for {
		select {
		case <-ticker2:
			puppetsync()
		case <-ticker:
			// http api check account setting
			fmt.Println("it's ok")
		}
	}
}
func puppetsync() {
	out, err := exec.Command("/usr/bin/sudo",
		"/usr/bin/puppet", "-t").Output()
	fmt.Print(out)
	if err != nil {
		fmt.Print(err)
	}
}
