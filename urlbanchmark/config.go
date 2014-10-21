package main

import (
	"encoding/json"
	"os"
)

// Config is metrictools config struct
type Setting struct {
	Tasks []Task `json:"tasks"`
}

type Task struct {
	Name               string `json:"name"`
	NotifyEmailAddress string `json:"notify_email_address"`
	EmailAddresses     string `json:"email_addresses"`
	Fails              int    `json:"fails_rise"`
	Success            int    `json:"success_rise"`
	Url                string `json:"url"`
	Interval           int    `json:"interval"`
	FailedCount        int    `json:"-"`
	SuccessCount       int    `json:"-"`
	State              bool   `json:"-"`
}

// ReadConfig used to read json to config
func ReadConfig(file string) (*Setting, error) {
	configFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()
	var setting Setting
	if err := json.NewDecoder(configFile).Decode(&setting); err != nil {
		return nil, err
	}
	return &setting, err
}
