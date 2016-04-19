package main

type SmartInfo struct {
	Hostname     string `json:"-"`
	Postion      string `json:"-"`
	DiskType     string `json:"DiskType"`
	Vendor       string `json:"Vendor"`
	Model        string `json:"Model"`
	UserCapacity string `json:"UserCapacity"`
	SerialNumber string `json:"SerialNumber"`
	Health       string `json:"Health"`
	Temperature  int    `json:"DiskType"`
	DefectBlocks int    `json:"DefectBlocks"`
	PowerOnHours int    `json:"PowerOnHours"`
}

func (d *SmartInfo) GetTags() map[string]string {
	tags := make(map[string]string)
	tags["Hostname"] = d.Hostname
	tags["Postion"] = d.Postion
	tags["DiskType"] = d.DiskType
	tags["Vendor"] = d.Vendor
	tags["Model"] = d.Model
	tags["UserCapacity"] = d.UserCapacity
	tags["SerialNumber"] = d.SerialNumber
	tags["Health"] = d.Health
	return tags
}
func (d *SmartInfo) GetFields() map[string]interface{} {
	fields := make(map[string]interface{})
	fields["Temperature"] = d.Temperature
	fields["DefectBlocks"] = d.DefectBlocks
	fields["PowerOnHours"] = d.PowerOnHours
	return fields
}
