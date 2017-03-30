package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

var (
	config = flag.String("c", "ddosapi.json", "config file")
)
var logger *log.Logger

func main() {
	flag.Parse()
	setting, err := readconfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	if len(setting["log_file"]) == 0 {
		setting["log_file"] = "./"
	}
	file, err := os.OpenFile(setting["log_file"], os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer file.Close()
	logger = log.New(file, "DDoSAPI:", log.Ldate|log.Ltime)
	API := &DDoSAPI{}
	API.Setting = setting
	go API.Run()
	r := gin.Default()
	//r.Use(s.loginFilter())
	r.Use(CORSMiddleware())
	authorized := r.Group("/api/v1")
	authorized.GET("/*filepath", API.APIGet)
	authorized.POST("/*filepath", API.APISet)
	r.Run(setting["port"])

	API.Stop()
}
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	}
}
