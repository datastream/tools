package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	config = flag.String("c", "blockapi.json", "config file")
)

var setting map[string]string

func main() {
	flag.Parse()
	var err error
	setting, err = readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	DA := &DDoSAgent{}
	DA.Setting = setting
	DA.Run()
	defer DA.Stop()
	r := gin.Default()
	//r.Use(DA.loginFilter())
	authorized := r.Group("/api/v1")
	authorized.GET("/ipset/{topic}", DA.showIPSet)
	authorized.GET("/nginx/{topic}", DA.showNginx)
	authorized.GET("/status/{serviceType}/{topic}", DA.Status)
	err = r.Run(DA.Setting["listen_addr"])
	if err != nil {
		log.Fatal(err)
	}
}
