package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func readconfig(file string) (map[string]string, error) {
	config_file, err := os.Open(file)
	config, err := ioutil.ReadAll(config_file)
	if err != nil {
		return nil, err
	}
	config_file.Close()
	setting := make(map[string]string)
	if err := json.Unmarshal(config, &setting); err != nil {
		return nil, err
	}
	return setting, nil
}
