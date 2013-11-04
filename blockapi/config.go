package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func readconfig(file string) (map[string][]string, error) {
	configFile, err := os.Open(file)
	config, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	configFile.Close()
	setting := make(map[string][]string)
	if err := json.Unmarshal(config, &setting); err != nil {
		return nil, err
	}
	return setting, nil
}
