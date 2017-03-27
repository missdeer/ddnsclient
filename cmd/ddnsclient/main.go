package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dfordsoft/ddnsclient/models"
)

type Setting struct {
	BasicAuthItems  []models.BasicAuthConfigurationItem  `json:"basic"`
	DnspodItems     []models.DnspodConfigurationItem     `json:"dnspod"`
	CloudflareItems []models.CloudflareConfigurationItem `json:"cloudflare"`
	CloudXNSItems   []models.CloudXNSConfigurationItem   `json:"cloudxns"`
}

var (
	insecureSkipVerify bool
	ifconfigURL        string
	currentExternalIP  string
	lastExternalIP     string
	dnspodDomainList   = &models.DnspodDomainList{}
)

func getCurrentExternalIP() (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
	}
	req, err := http.NewRequest("GET", ifconfigURL, nil)
	if err != nil {
		fmt.Println("create request to ifconfig failed", err)
		return "", err
	}
	req.Header.Set("User-Agent", "curl/7.41.0")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request %s failed", ifconfigURL)
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

	return "", errors.New("invalid IP address: " + string(body))
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
		fmt.Printf("reading cloudflare all domain list failed\n")
		return -1
	}

	recordList := new(models.CloudXNSDomainList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling CloudXNS all domain list %s failed: %v\n", string(body), err)
		return -1
	}

	docoratedDomain := domain + "."
	for _, v := range recordList.Data {
		if v.Domain == docoratedDomain {
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

	recordList := new(models.CloudXNSHostRecordList)
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

	// get resolve record list
	cloudxnsAPIUrl := fmt.Sprintf("https://www.cloudxns.net/api2/record/%d?host_id=%d&offset=0&row_num=2000", domainId, hostRecordId)
	req, err := http.NewRequest("GET", cloudxnsAPIUrl, nil)
	req.Header.Set("API-KEY", apiKey)
	apiRequestDate := time.Now().String()
	req.Header.Add("API-REQUEST-DATE", apiRequestDate)
	sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + apiRequestDate + secretKey))
	req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Getting CloudXNS resolve record list failed", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("reading cloudflare all resolve records failed\n")
		return err
	}

	recordList := new(models.CloudXNSResolveList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling CloudXNS all resolve records %s failed: %v\n", string(body), err)
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

	postValues := make(map[string]interface{})
	if foundRecord {
		// update
		postValues["domain_id"] = domainId
		postValues["host"] = sub_domain
		postValues["value"] = currentExternalIP
		p, err := json.Marshal(postValues)
		if err != nil {
			fmt.Println("marshal update body failed", err)
			return err
		}

		cloudxnsAPIUrl := fmt.Sprintf("https://www.cloudxns.net/api2/record/%d", recordId)
		req, err := http.NewRequest("PUT", cloudxnsAPIUrl, bytes.NewReader(p))
		req.Header.Set("API-KEY", apiKey)
		apiRequestDate := time.Now().String()
		req.Header.Add("API-REQUEST-DATE", apiRequestDate)
		sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + string(p) + apiRequestDate + secretKey))
		req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[%v] Updating CloudXNS resolve item failed", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("A record updated to cloudXNS: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
	} else {
		// insert
		postValues["domain_id"] = fmt.Sprintf("%d", domainId)
		postValues["host"] = sub_domain
		postValues["value"] = currentExternalIP
		postValues["type"] = "A"
		postValues["line_id"] = fmt.Sprintf("%d", lineId)
		p, err := json.Marshal(postValues)
		if err != nil {
			fmt.Println("marshal update body failed", err)
			return err
		}
		cloudxnsAPIUrl := "https://www.cloudxns.net/api2/record"
		req, err := http.NewRequest("POST", cloudxnsAPIUrl, bytes.NewReader(p))
		req.Header.Set("API-KEY", apiKey)
		apiRequestDate := time.Now().String()
		req.Header.Add("API-REQUEST-DATE", apiRequestDate)
		sum := md5.Sum([]byte(apiKey + cloudxnsAPIUrl + string(p) + apiRequestDate + secretKey))
		req.Header.Add("API-HMAC", hex.EncodeToString(sum[:]))
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[%v] inserting CloudXNS resolve item failed", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("A record inserted to cloudXNS: %s.%s => %s\n", sub_domain, domain, currentExternalIP)
	}
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

	recordList := new(models.CloudflareRecordList)
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
		respBody := new(models.CloudflareNewRecordResponseBody)
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

func dnspodRequestByToken(id string, token string, domain string, sub_domain string) error {
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
			"login_token": {id + "," + token},
			"format":      {"json"},
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
		"login_token": {id + "," + token},
		"format":      {"json"},
		"domain_id":   {strconv.Itoa(domainId)},
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

	recordList := new(models.DnspodRecordList)
	if err = json.Unmarshal(body, recordList); err != nil {
		fmt.Printf("unmarshalling record list %s failed\n", string(body))
		fmt.Println(err)
		return err
	}
	foundRecord := false
	var recordID string
	for _, v := range recordList.Records {
		if v.Name == sub_domain {
			foundRecord = true
			recordID = v.Id
			break
		}
	}

	if foundRecord == false {
		// if the sub domain doesn't exist, add one
		addRecordURL := "https://dnsapi.cn/Record.Create"
		resp, err := client.PostForm(addRecordURL, url.Values{
			"login_token": {id + "," + token},
			"format":      {"json"},
			"domain_id":   {strconv.Itoa(domainId)},
			"sub_domain":  {sub_domain},
			"record_type": {"A"},
			"record_line": {"默认"},
			"value":       {currentExternalIP},
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
		modifyRecordURL := "https://dnsapi.cn/Record.Modify"
		resp, err := client.PostForm(modifyRecordURL, url.Values{
			"login_token": {id + "," + token},
			"format":      {"json"},
			"record_id":   {recordID},
			"domain_id":   {strconv.Itoa(domainId)},
			"sub_domain":  {sub_domain},
			"record_type": {"A"},
			"record_line": {"默认"},
			"value":       {currentExternalIP},
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

	recordList := new(models.DnspodRecordList)
	if err = json.Unmarshal(body, recordList); err != nil {
		fmt.Printf("unmarshalling record list %s failed\n", string(body))
		fmt.Println(err)
		return err
	}
	foundRecord := false
	var recordID string
	for _, v := range recordList.Records {
		if v.Name == sub_domain {
			foundRecord = true
			recordID = v.Id
			break
		}
	}

	if foundRecord == false {
		// if the sub domain doesn't exist, add one
		addRecordURL := "https://dnsapi.cn/Record.Create"
		resp, err := client.PostForm(addRecordURL, url.Values{
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
		modifyRecordURL := "https://dnsapi.cn/Record.Modify"
		resp, err := client.PostForm(modifyRecordURL, url.Values{
			"login_email":    {user},
			"login_password": {password},
			"format":         {"json"},
			"record_id":      {recordID},
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

	basicAuth := func(v models.BasicAuthConfigurationItem) {
		for {
			if err := basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}

	dnspod := func(v models.DnspodConfigurationItem) {
		for {
			if len(v.Token) != 0 && len(v.TokenId) != 0 {
				if err := dnspodRequestByToken(v.TokenId, v.Token, v.Domain, v.SubDomain); err == nil {
					break
				}
			} else if len(v.UserName) != 0 && len(v.Password) != 0 {
				if err := dnspodRequest(v.UserName, v.Password, v.Domain, v.SubDomain); err == nil {
					break
				}
			}
			time.Sleep(1 * time.Minute)
		}
	}

	cloudflare := func(v models.CloudflareConfigurationItem) {
		for {
			if err := cloudflareRequest(v.UserName, v.Token, v.Domain, v.SubDomain); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}

	cloudxns := func(v models.CloudXNSConfigurationItem) {
		for {
			if err := cloudxnsRequest(v.APIKey, v.SecretKey, v.Domain, v.SubDomain); err == nil {
				break
			}
			time.Sleep(1 * time.Minute)
		}
	}
	if len(currentExternalIP) != 0 && lastExternalIP != currentExternalIP {
		for _, v := range setting.BasicAuthItems {
			go basicAuth(v)
		}

		for _, v := range setting.DnspodItems {
			go dnspod(v)
		}

		for _, v := range setting.CloudflareItems {
			go cloudflare(v)
		}

		for _, v := range setting.CloudXNSItems {
			go cloudxns(v)
		}
		lastExternalIP = currentExternalIP
	}
}

var conf string

func main() {
	flag.BoolVar(&insecureSkipVerify, "insecureSkipVerify", false, "if true, TLS accepts any certificate")
	flag.StringVar(&ifconfigURL, "ifconfig", "https://if.yii.li", "set ifconfig URL")
	flag.StringVar(&conf, "config", "app.conf", "set application config")
	flag.Parse()

	fmt.Println("Dynamic DNS client")
	appConf, err := os.Open(conf)
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
}
