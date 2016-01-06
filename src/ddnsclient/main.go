package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BasicAuthConfigurationItem struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	Url      string `json:"url"`
}

type DnspodConfigurationItem struct {
	UserName  string `json:"username"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
}

type CloudflareConfigurationItem struct {
	UserName  string `json:"username"`
	Token     string `json:"token"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
}

type CloudXNSConfigurationItem struct {
	APIKey    string `json:"apikey"`
	SecretKey string `json:"secretkey"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
}

type Setting struct {
	BasicAuthItems  []BasicAuthConfigurationItem  `json:"basic"`
	DnspodItems     []DnspodConfigurationItem     `json:"dnspod"`
	CloudflareItems []CloudflareConfigurationItem `json:"cloudflare"`
	CloudXNSItems   []CloudXNSConfigurationItem   `json:"cloudxns"`
}

type CloudXNSDomainItem struct {
	Id             int       `json:"id,string"`
	Domain         string    `json:"domain"`
	Status         string    `json:"status"`
	AuditStatus    string    `json:"audit_status"`
	TakeOverStatus string    `json:"take_over_status"`
	Level          int       `json:"level,string"`
	CreateTime     time.Time `json:"create_time"`
	UpdateTime     time.Time `json:"update_time"`
	TTL            int       `json:"ttl,string"`
}

type CloudXNSDomainList struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Total   string               `json:"total"`
	Data    []CloudXNSDomainItem `json:"data"`
}

type CloudXNSHostRecordItem struct {
	Id         int    `json:"id,string"`
	Host       string `json:"host"`
	RecordNum  int    `json:"record_num,string"`
	DomainName string `json:"domain_name"`
}

type CloudXNSHostRecordList struct {
	Code    string                   `json:"code"`
	Message string                   `json:"message"`
	Total   string                   `json:"total"`
	Data    []CloudXNSHostRecordItem `json:"data"`
}

type CloudXNSResolveItem struct {
	RecordId   int         `json:"record_id,string"`
	HostId     int         `json:"host_id,string"`
	Host       int         `json:"host,string"`
	LineZh     string      `json:"line_zh"`
	LineEn     string      `json:"line_en"`
	LineId     int         `json:"line_id"`
	MX         interface{} `json:"mx"`
	Value      string      `json:"value"`
	Type       string      `json:"type"`
	Status     string      `json:"status"`
	CreateTime time.Time   `json:"create_time"`
	UpdateTime time.Time   `json:"update_time"`
}

type CloudXNSResolveList struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Total   string                `json:"total"`
	Data    []CloudXNSResolveItem `json:"data"`
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

var (
	currentExternalIP string
	lastExternalIP    string
	dnspodDomainList  = &DnspodDomainList{}
)

func getCurrentExternalIP() (string, error) {
	client := &http.Client{}
	ifconfigUrl := "https://if.yii.li"
	req, err := http.NewRequest("GET", ifconfigUrl, nil)
	req.Header.Set("User-Agent", "curl/7.41.0")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed", ifconfigUrl)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading ifconfig response failed\n")
		return "", err
	}

	for i := len(body); i > 0 && (body[i-1] < '0' || body[i-1] > '9'); i = len(body) {
		body = body[:i-1]
	}

	if matched, err := regexp.Match(`^([0-9]{1,3}\.){3,3}[0-9]{1,3}$`, body); err == nil && matched == true {
		return string(body), nil
	}

	return "", errors.New("invalid IP address")
}

func basicAuthorizeHttpRequest(user string, password string, requestUrl string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.SetBasicAuth(user, password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed\n", requestUrl)
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading response failed\n")
		return err
	}
	return nil
}

func cloudxnsFindDomain(apiKey string, secretKey string, domain string) int {
	client := &http.Client{}
	// get domain list
	cloudxnsAPIUrl := "https://www.cloudxns.net/api2/domain"
	req, err := http.NewRequest("GET", cloudxnsAPIUrl, nil)
	req.Header.Set("API-KEY", apiKey)
	apiRequestDate := time.Now().String()
	req.Header.Add("API-REQUEST-DATE", apiRequestDate)
	sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + apiRequestDate + secretKey))
	req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Getting CloudXNS domain list failed", err)
		return -1
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare all records failed\n")
		return -1
	}

	recordList := new(CloudXNSDomainList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling CloudXNS all records %s failed\n", string(body))
		return -1
	}

	for _, v := range recordList.Data {
		if v.Domain == domain {
			return v.Id
		}
	}
	return -1
}

func cloudxnsFindHostRecord(apiKey string, secretKey string, domainId int, sub_domain string) int {
	client := &http.Client{}
	// get host record list
	cloudxnsAPIUrl := fmt.Sprintf("https://www.cloudxns.net/api2/host/%d?offset=0&row_num=2000", domainId)
	req, err := http.NewRequest("GET", cloudxnsAPIUrl, nil)
	req.Header.Set("API-KEY", apiKey)
	apiRequestDate := time.Now().String()
	req.Header.Add("API-REQUEST-DATE", apiRequestDate)
	sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + apiRequestDate + secretKey))
	req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Getting CloudXNS host record list failed", err)
		return -1
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading CloudXNS all host records failed\n")
		return -1
	}

	recordList := new(CloudXNSHostRecordList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling CloudXNS all records %s failed\n", string(body))
		return -1
	}

	for _, v := range recordList.Data {
		if v.Host == sub_domain {
			return v.Id
		}
	}

	return -1
}

func cloudxnsRequest(apiKey string, secretKey string, domain string, sub_domain string) error {
	// find the domain
	domainId := cloudxnsFindDomain(apiKey, secretKey, domain)
	if domainId == -1 {
		fmt.Println("can't find domain in list", domain)
		return errors.New("domain not exists")
	}
	// find the host
	hostRecordId := cloudxnsFindHostRecord(apiKey, secretKey, domainId, sub_domain)
	if hostRecordId == -1 {
		fmt.Println("can't find host record in list", sub_domain)
		return errors.New("host record not exists")
	}
	// find the resolve record
	client := &http.Client{}

	// get domain list
	cloudxnsAPIUrl := fmt.Sprintf("https://www.cloudxns.net/api2/record/%d?host_id=%d&offset=0&row_num=2000", domainId, hostRecordId)
	req, err := http.NewRequest("GET", cloudxnsAPIUrl, nil)
	req.Header.Set("API-KEY", "")
	apiRequestDate := time.Now().String()
	req.Header.Add("API-REQUEST-DATE", apiRequestDate)
	sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + apiRequestDate + secretKey))
	req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Getting CloudXNS domain list failed", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare all records failed\n")
		return err
	}

	recordList := new(CloudXNSResolveList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling CloudXNS all records %s failed\n", string(body))
		return err
	}

	// insert or update
	foundRecord := false
	var recordId int
	var lineId int
	if len(recordList.Data) > 0 {
		foundRecord = true
		recordId = recordList.Data[0].RecordId
		lineId = recordList.Data[0].LineId
	}

	if foundRecord {
		// update
		postValues := url.Values{
			"domain_id": {fmt.Sprintf("%d", domainId)},
			"host":      {sub_domain},
			"value":     {currentExternalIP},
		}
		cloudxnsAPIUrl := fmt.Sprintf("https://www.cloudxns.net/api2/record/%d", recordId)

		req, err := http.NewRequest("PUT", cloudxnsAPIUrl, strings.NewReader(postValues.Encode()))
		req.Header.Set("API-KEY", apiKey)
		apiRequestDate := time.Now().String()
		req.Header.Add("API-REQUEST-DATE", apiRequestDate)
		sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + postValues.Encode() + apiRequestDate + secretKey))
		req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[%v] Updating CloudXNS resolve item failed", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
	} else {
		// insert
		postValues := url.Values{
			"domain_id": {fmt.Sprintf("%d", domainId)},
			"host":      {sub_domain},
			"value":     {currentExternalIP},
			"type":      {"A"},
			"line_id":   {fmt.Sprintf("%d", lineId)},
		}
		cloudxnsAPIUrl := "https://www.cloudxns.net/api2/record"
		req, err := http.NewRequest("POST", cloudxnsAPIUrl, strings.NewReader(postValues.Encode()))
		req.Header.Set("API-KEY", apiKey)
		apiRequestDate := time.Now().String()
		req.Header.Add("API-REQUEST-DATE", apiRequestDate)
		sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + postValues.Encode() + apiRequestDate + secretKey))
		req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[%v] inserting CloudXNS resolve item failed", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
	}
	fmt.Printf("A record updated to cloudXNS: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
	return nil
}

func cloudflareRequest(user string, token string, domain string, sub_domain string) error {
	// get domain all records
	cloudflareAPIUrl := "https://www.cloudflare.com/api_json.html"
	client := &http.Client{}
	resp, err := client.PostForm(cloudflareAPIUrl, url.Values{
		"a":     {"rec_load_all"},
		"tkn":   {token},
		"email": {user},
		"z":     {domain},
	})
	if err != nil {
		fmt.Printf("request cloudflare all records failed\n")
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare all records failed\n")
		return err
	}

	recordList := new(CloudflareRecordList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling cloudflare all records %s failed\n", string(body))
		return err
	}

	// insert or update
	foundRecord := false
	var recordId string
	for _, v := range recordList.Response.Recs.Objs {
		if v.Type == "A" && v.DisplayName == sub_domain {
			recordId = v.Id
			foundRecord = true
			break
		}
	}
	if foundRecord == false {
		// insert a new record
		resp, err := client.PostForm(cloudflareAPIUrl, url.Values{
			"a":       {"rec_new"},
			"tkn":     {token},
			"email":   {user},
			"z":       {domain},
			"ttl":     {"1"},
			"type":    {"A"},
			"name":    {sub_domain},
			"content": {currentExternalIP},
		})
		if err != nil {
			fmt.Printf("request cloudflare new record failed\n")
			return err
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("reading cloudflare new record failed\n")
			return err
		}
		// extract the new record id
		respBody := new(CloudflareNewRecordResponseBody)
		if err = json.Unmarshal(body, respBody); err != nil {
			fmt.Printf("unmarshalling cloudflare new record response body failed\n")
			return err
		}
		recordId = respBody.Response.Rec.Obj.Id
		fmt.Printf("[%v] A record inserted into cloudflare: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIP)
	}
	// update the record
	resp, err = client.PostForm(cloudflareAPIUrl, url.Values{
		"a":            {"rec_edit"},
		"tkn":          {token},
		"email":        {user},
		"z":            {domain},
		"type":         {"A"},
		"service_mode": {"0"},
		"ttl":          {"1"},
		"id":           {recordId},
		"name":         {sub_domain},
		"content":      {currentExternalIP},
	})
	if err != nil {
		fmt.Printf("request cloudflare records edit failed\n")
		return err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare record edit response failed\n")
		return err
	}
	fmt.Printf("[%v] A record updated to cloudflare: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIP)
	return nil
}

func dnspodRequest(user string, password string, domain string, sub_domain string) error {
	needDomainList := false
	if len(dnspodDomainList.Domains) == 0 {
		needDomainList = true
	}
	var domainId int = 0
	if needDomainList == false {
		needDomainList = true
		for _, v := range dnspodDomainList.Domains {
			if v.Name == domain {
				needDomainList = false
				domainId = v.Id
				break
			}
		}
	}

	client := &http.Client{}
	if needDomainList {
		// get domainn id first
		domainListUrl := "https://dnsapi.cn/Domain.List"
		resp, err := client.PostForm(domainListUrl, url.Values{
			"login_email":    {user},
			"login_password": {password},
			"format":         {"json"},
		})
		if err != nil {
			fmt.Printf("request domain list failed\n")
			return err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("reading domain list failed\n")
			return err
		}

		if err = json.Unmarshal(body, &dnspodDomainList); err != nil {
			fmt.Printf("unmarshalling domain list %s failed\n", string(body))
			return err
		}
	}
	foundDomain := false
	for _, v := range dnspodDomainList.Domains {
		if v.Name == domain {
			foundDomain = true
			domainId = v.Id
			break
		}
	}

	if foundDomain == false {
		fmt.Printf("domain %s doesn't exists\n", domain)
		return errors.New("domain not found")
	}

	// check record list
	recordListUrl := "https://dnsapi.cn/Record.List"
	resp, err := client.PostForm(recordListUrl, url.Values{
		"login_email":    {user},
		"login_password": {password},
		"format":         {"json"},
		"domain_id":      {strconv.Itoa(domainId)},
	})
	if err != nil {
		fmt.Printf("request record list failed\n")
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading record list failed\n")
		return err
	}

	recordList := new(DnspodRecordList)
	if err = json.Unmarshal(body, recordList); err != nil {
		fmt.Printf("unmarshalling record list %s failed\n", string(body))
		fmt.Println(err)
		return err
	}
	foundRecord := false
	var recordId string
	for _, v := range recordList.Records {
		if v.Name == sub_domain {
			foundRecord = true
			recordId = v.Id
			break
		}
	}

	if foundRecord == false {
		// if the sub domain doesn't exist, add one
		addRecordUrl := "https://dnsapi.cn/Record.Create"
		resp, err := client.PostForm(addRecordUrl, url.Values{
			"login_email":    {user},
			"login_password": {password},
			"format":         {"json"},
			"domain_id":      {strconv.Itoa(domainId)},
			"sub_domain":     {sub_domain},
			"record_type":    {"A"},
			"record_line":    {"默认"},
			"value":          {currentExternalIP},
		})
		if err != nil {
			fmt.Printf("request record insert failed\n")
			return err
		}
		defer resp.Body.Close()

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record insert response failed\n")
			return err
		}

		fmt.Printf("[%v] A record inserted into DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIP)
	} else {
		// otherwise just update it
		modifyRecordUrl := "https://dnsapi.cn/Record.Modify"
		resp, err := client.PostForm(modifyRecordUrl, url.Values{
			"login_email":    {user},
			"login_password": {password},
			"format":         {"json"},
			"record_id":      {recordId},
			"domain_id":      {strconv.Itoa(domainId)},
			"sub_domain":     {sub_domain},
			"record_type":    {"A"},
			"record_line":    {"默认"},
			"value":          {currentExternalIP},
		})
		if err != nil {
			fmt.Printf("request record modify failed\n")
			return err
		}
		defer resp.Body.Close()

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record modify response failed\n")
			return err
		}
		fmt.Printf("[%v] A record updated to DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIP)
	}

	return nil
}

func updateDDNS(setting *Setting) {
	var err error
	currentExternalIP, err = getCurrentExternalIP()
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(currentExternalIP) != 0 && lastExternalIP != currentExternalIP {
		for _, v := range setting.BasicAuthItems {
			retried := false
		start_basicauth:
			if err = basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url); err != nil {
				time.Sleep(5 * time.Second)
				if !retried {
					retried = true
					goto start_basicauth
				}
			}
		}

		for _, v := range setting.DnspodItems {
			retried := false
		start_dnspod:
			if err = dnspodRequest(v.UserName, v.Password, v.Domain, v.SubDomain); err != nil {
				time.Sleep(5 * time.Second)
				if !retried {
					retried = true
					goto start_dnspod
				}
			}
		}

		for _, v := range setting.CloudflareItems {
			retried := false
		start_cloudflare:
			if err = cloudflareRequest(v.UserName, v.Token, v.Domain, v.SubDomain); err != nil {
				time.Sleep(5 * time.Second)
				if !retried {
					retried = true
					goto start_cloudflare
				}
			}
		}

		for _, v := range setting.CloudXNSItems {
			retried := false
		start_cloudxns:
			if err = cloudxnsRequest(v.APIKey, v.SecretKey, v.Domain, v.SubDomain); err != nil {
				time.Sleep(5 * time.Second)
				if !retried {
					retried = true
					goto start_cloudxns
				}
			}
		}
		lastExternalIP = currentExternalIP
	}
}

func main() {
	fmt.Println("Dynamic DNS client")

	appConf, err := os.Open("app.conf")
	if err != nil {
		fmt.Println("opening app.conf failed:", err)
		return
	}

	defer func() {
		appConf.Close()
	}()

	b, err := ioutil.ReadAll(appConf)
	if err != nil {
		fmt.Println("reading app.conf failed:", err)
		return
	}
	setting := new(Setting)
	err = json.Unmarshal(b, &setting)
	if err != nil {
		fmt.Println("unmarshalling app.conf failed:", err)
		return
	}

	go updateDDNS(setting)
	timer := time.NewTicker(1 * time.Minute) // every 1 minute
	for {
		select {
		case <-timer.C:
			go updateDDNS(setting)
		}
	}
	timer.Stop()
}
