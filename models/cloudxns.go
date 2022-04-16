package models

type CloudXNSConfigurationItem struct {
	APIKey    string `json:"apikey"`
	SecretKey string `json:"secretkey"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
	Internal  bool   `json:",omitempty"`
}

type CloudXNSDomainItem struct {
	Id             int    `json:"id,string"`
	Domain         string `json:"domain"`
	Status         string `json:"status"`
	AuditStatus    string `json:"audit_status"`
	TakeOverStatus string `json:"take_over_status"`
	Level          int    `json:"level,string"`
	CreateTime     string `json:"create_time"`
	UpdateTime     string `json:"update_time"`
	TTL            int    `json:"ttl,string"`
}

type CloudXNSDomainList struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Total   int                  `json:"total,string"`
	Data    []CloudXNSDomainItem `json:"data"`
}

type CloudXNSHostRecordItem struct {
	Id         int    `json:"id,string"`
	Host       string `json:"host"`
	RecordNum  int    `json:"record_num,string"`
	DomainName string `json:"domain_name"`
}

type CloudXNSHostRecordList struct {
	Code    int                      `json:"code"`
	Message string                   `json:"message"`
	Total   int                      `json:"total,string"`
	Data    []CloudXNSHostRecordItem `json:"hosts"`
}

type CloudXNSResolveItem struct {
	RecordId   int         `json:"record_id,string"`
	HostId     int         `json:"host_id,string"`
	Host       string      `json:"host"`
	LineZh     string      `json:"line_zh"`
	LineEn     string      `json:"line_en"`
	LineId     int         `json:"line_id,string"`
	MX         interface{} `json:"mx"`
	Value      string      `json:"value"`
	Type       string      `json:"type"`
	Status     string      `json:"status"`
	CreateTime string      `json:"create_time"`
	UpdateTime string      `json:"update_time"`
}

type CloudXNSResolveList struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Total   int                   `json:"total"`
	Data    []CloudXNSResolveItem `json:"data"`
}
