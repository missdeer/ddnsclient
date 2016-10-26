package models

type CloudflareConfigurationItem struct {
	UserName  string `json:"username"`
	Token     string `json:"token"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
}

type CloudflareRecordItem struct {
	Id          string `json:"rec_id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type CloudflareRecords struct {
	Objs []CloudflareRecordItem `json:"objs"`
}

type CloudflareResponse struct {
	Recs CloudflareRecords `json:"recs"`
}

type CloudflareRecordList struct {
	Response CloudflareResponse `json:"response"`
}

type CloudflareNewRecords struct {
	Obj CloudflareRecordItem `json:"obj"`
}

type CloudflareNewRecordResponse struct {
	Rec CloudflareNewRecords `json:"rec"`
}

type CloudflareNewRecordResponseBody struct {
	Response CloudflareNewRecordResponse `json:"response"`
}
