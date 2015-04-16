package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

type Setting struct {
	BasicAuthItems  []BasicAuthConfigurationItem  `json:"basic"`
	DnspodItems     []DnspodConfigurationItem     `json:"dnspod"`
	CloudflareItems []CloudflareConfigurationItem `json:"cloudflare"`
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
	ifconfigUrl := "https://ifconfig.minidump.info"
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
	return string(body), nil
}

func basicAuthorizeHttpRequest(user string, password string, requestUrl string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.SetBasicAuth(user, password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed\n", requestUrl)
		return
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading response failed\n")
		return
	}
}

func cloudflareRequest(user string, token string, domain string, sub_domain string) {
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
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare all records failed\n")
		return
	}

	recordList := new(CloudflareRecordList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling cloudflare all records %s failed\n", string(body))
		return
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
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("reading cloudflare new record failed\n")
			return
		}
		// extract the new record id
		respBody := new(CloudflareNewRecordResponseBody)
		if err = json.Unmarshal(body, respBody); err != nil {
			fmt.Printf("unmarshalling cloudflare new record response body failed\n")
			return
		}
		recordId = respBody.Response.Rec.Obj.Id
		fmt.Printf("A record inserted into cloudflare: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
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
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare record edit response failed\n")
		return
	}
	fmt.Printf("A record updated to cloudflare: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
}

func dnspodRequest(user string, password string, domain string, sub_domain string) {
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
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("reading domain list failed\n")
			return
		}

		if err = json.Unmarshal(body, &dnspodDomainList); err != nil {
			fmt.Printf("unmarshalling domain list %s failed\n", string(body))
			return
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
		return
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
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading record list failed\n")
		return
	}

	recordList := new(DnspodRecordList)
	if err = json.Unmarshal(body, recordList); err != nil {
		fmt.Printf("unmarshalling record list %s failed\n", string(body))
		fmt.Println(err)
		return
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
			return
		}
		defer resp.Body.Close()

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record insert response failed\n")
			return
		}

		fmt.Printf("A record inserted into DNSPOD: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
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
			return
		}
		defer resp.Body.Close()

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record modify response failed\n")
			return
		}
		fmt.Printf("A record updated to DNSPOD: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
	}
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
			basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url)
		}

		for _, v := range setting.DnspodItems {
			dnspodRequest(v.UserName, v.Password, v.Domain, v.SubDomain)
		}
		lastExternalIP = currentExternalIP

		for _, v := range setting.CloudflareItems {
			cloudflareRequest(v.UserName, v.Token, v.Domain, v.SubDomain)
		}
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
	timer := time.NewTicker(time.Duration(1) * time.Minute) // every 1 minute
	for {
		select {
		case <-timer.C:
			go updateDDNS(setting)
		}
	}
	timer.Stop()
}
