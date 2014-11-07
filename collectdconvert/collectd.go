package main

import (
	"regexp"
)

// CollectdJSON is collectd's json data format
type CollectdJSON struct {
	Values         []float64 `json:"values"`
	DataSetTypes   []string  `json:"dstypes"`
	DataSetNames   []string  `json:"dsnames"`
	Timestamp      float64   `json:"time"`
	Interval       float64   `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
}
func (c *CollectdJSON) GetName() string {
	return c.Plugin
}

func (c *CollectdJSON) GetMetricName(index int) string {
	metricName := c.Host + "_" + c.Plugin
	if len(c.PluginInstance) > 0 {
		if matched, _ := regexp.MatchString(`^\d+$`, c.PluginInstance); matched {
			metricName += c.PluginInstance
		} else {
			metricName += "_" + c.PluginInstance
		}
	}
	tSize := len(c.Type)
	pSize := len(c.Plugin)
	if tSize > 0 {
		if pSize <= tSize && c.Type[:pSize] == c.Plugin {
			metricName += c.Type[pSize:]
		} else {
			metricName += "." + c.Type
		}
	}
	if len(c.TypeInstance) > 0 {
		if matched, _ := regexp.MatchString(`^\d+$`, c.TypeInstance); matched {
			metricName += c.TypeInstance
		} else {
			metricName += "_" + c.TypeInstance
		}
	}
	if c.DataSetNames[index] != "value" {
		metricName += "." + c.DataSetNames[index]
	}
	return metricName
}

func (c *CollectdJSON) GetMetricRate(value float64, timestamp int64, index int) float64 {
	var nValue float64
	if c.DataSetTypes[index] == "counter" || c.DataSetTypes[index] == "derive" {
		nValue = (c.Values[index] - value) / float64(int64(c.Timestamp)-timestamp)
		if nValue < 0 {
			nValue = 0
		}
	} else {
		nValue = c.Values[index]
	}
	return nValue
}
