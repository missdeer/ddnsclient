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

type BasicAuthItem struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	Url      string `json:"url"`
}

type DnspodItem struct {
	UserName  string `json:"username"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	SubDomain string `json:"sub_domain"`
}

type Setting struct {
	BasicAuthItems []BasicAuthItem `json:"basic"`
	DnspodItems    []DnspodItem    `json:"dnspod"`
}

type DomainItem struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type DomainList struct {
	Domains []DomainItem `json:"domains"`
}

type RecordItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type RecordList struct {
	Records []RecordItem `json:"records"`
}

var (
	currentExternalIP string
	lastExternalIP    string
	domainList        = &DomainList{}
)

func getCurrentExternalIP() string {
	return ""
}

func basicAuthorizeHttpRequest(user string, password string, requestUrl string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.SetBasicAuth(user, password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed", requestUrl)
		return
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading response failed")
		return
	}
}

func dnspodRequest(user string, password string, domain string, sub_domain string) {
	needDomainList := false
	if len(domainList.Domains) == 0 {
		needDomainList = true
	}
	var domainId int = 0
	if needDomainList == false {
		needDomainList = true
		for _, v := range domainList.Domains {
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
			fmt.Printf("request domain list failed")
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("reading domain list failed")
			return
		}

		if err = json.Unmarshal(body, &domainList); err != nil {
			fmt.Printf("unmarshalling domain list %s failed", string(body))
			return
		}
	}
	foundDomain := false
	for _, v := range domainList.Domains {
		if v.Name == domain {
			foundDomain = true
			domainId = v.Id
			break
		}
	}

	if foundDomain == false {
		fmt.Printf("domain %s doesn't exists", domain)
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
		fmt.Printf("request record list failed")
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading record list failed")
		return
	}

	recordList := new(RecordList)
	if err = json.Unmarshal(body, recordList); err != nil {
		fmt.Printf("unmarshalling record list %s failed", string(body))
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
	fmt.Println("4")
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
			fmt.Printf("request record insert failed")
			return
		}

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record insert response failed")
			return
		}

		fmt.Printf("A record inserted: %s.%s => %s", sub_domain, domain, getCurrentExternalIP())
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
			fmt.Printf("request record modify failed")
			return
		}

		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Printf("reading record modify response failed")
			return
		}
		fmt.Printf("A record updated: %s.%s => %s", sub_domain, domain, getCurrentExternalIP())
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
	setting := &Setting{}
	err = json.Unmarshal(b, &setting)
	if err != nil {
		fmt.Println("unmarshalling app.conf failed:", err)
		return
	}

	timer := time.NewTicker(time.Duration(1) * time.Minute) // every 1 minute
	for {
		select {
		case <-timer.C:
			currentExternalIP = getCurrentExternalIP()
			if lastExternalIP != currentExternalIP {
				for _, v := range setting.BasicAuthItems {
					basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url)
				}

				for _, v := range setting.DnspodItems {
					dnspodRequest(v.UserName, v.Password, v.Domain, v.SubDomain)
				}
				lastExternalIP = currentExternalIP
			}
		}
	}
	timer.Stop()
}
