package main

import (
        "flag"
	"github.com/gin-gonic/gin"
        "fmt"
        "net/http"
        "os"
	"io/ioutil"
        "strings"
)

var (
        cachePath = flag.String("d", "./", "cache files dir")
        listenAddr = flag.String("l", "8080", "listen addr")
)
func main() {
	flag.Parse()
	r := gin.Default()
	r.GET("/*", DownloadDtD)
        r.Run(*listenAddr)
}

func DownloadDtD(c *gin.Context) {
	c.Header("Content-Type", "text/xml; charset=\"utf-8\"")
	url := c.Request.URL.RequestURI()[1:]
	resp, err := http.Get(fmt.Sprintf("http://%s",url))
	if resp.StatusCode != 200 {
		c.String(http.StatusServiceUnavailable, "fail to reach baclend")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "fail to read body")
	}
	defer resp.Body.Close()
	c.Header("Cache-Control","max-age=436800")
	c.String(http.StatusOK, string(body))
	filePath := fmt.Sprintf("%s%s", *cachePath, c.Request.URL.Path)
	paths := strings.Split(filePath, "/")
	err = os.MkdirAll(strings.Join(paths[:len(paths)-1], "/"), os.ModeDir)
	fp, err := os.Open(filePath)
	if err != nil {
		return
	}
	n, err := fp.Write(body)
	if err != nil || len(body) != n {
		os.Remove(filePath)
	}
	defer fp.Close()
}
