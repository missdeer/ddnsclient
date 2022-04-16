package models

type DnspodConfigurationItem struct {
	TokenId   string `json:"id"`
	Token     string `json:"token"`
	UserName  string `json:"username"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
	Internal  bool   `json:",omitempty"`
}

type DnspodDomainItem struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type DnspodDomainList struct {
	Domains []DnspodDomainItem `json:"domains"`
}

type DnspodRecordItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type DnspodRecordList struct {
	Records []DnspodRecordItem `json:"records"`
}
