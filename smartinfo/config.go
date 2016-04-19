package main

import (
	"encoding/json"
	"github.com/goinggo/mapstructure"
	"io/ioutil"
	"os"
)

// Config is metrictools config struct
type Setting struct {
	LookupdAddresses []string `jpath:"lookupd_addresses"`
	Topic            string   `jpath:"topic"`
	Channel          string   `jpath:"channel"`
	InfluxdbAddress  string   `jpath:"influxdb_address"`
	InfluxdbUser     string   `jpath:"influxdb_user"`
	InfluxdbPassword string   `jpath:"influxdb_password"`
	InfluxdbDatabase string   `jpath:"influxdb_database"`
	MaxInFlight      int      `jpath:"maxinflight"`
}

// ReadConfig used to read json to config
func ReadConfig(file string) (*Setting, error) {
	configFile, err := os.Open(file)
	config, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	configFile.Close()
	docMap := make(map[string]interface{})
	if err := json.Unmarshal(config, &docMap); err != nil {
		return nil, err
	}
	setting := &Setting{}
	err = mapstructure.DecodePath(docMap, setting)
	return setting, err
}
