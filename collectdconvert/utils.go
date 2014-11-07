package main

// MetricData
type Metric struct {
	Name          string
	LastTimestamp int64
	LastValue     float64
	RateValue     float64
	Mtype         string
}

func KeyValueEncode(key int64, value float64) ([]byte, error) {
	kv := &KeyValue{
		Timestamp: proto.Int64(key),
		Value:     proto.Float64(value),
	}
	record, err := proto.Marshal(kv)
	return record, err
}

func KeyValueDecode(record []byte) (KeyValue, error) {
	var kv KeyValue
	err := proto.Unmarshal(record, &kv)
	return kv, err
}
