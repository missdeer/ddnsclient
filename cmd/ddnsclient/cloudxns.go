package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/missdeer/ddnsclient/models"
)

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

func cloudxnsFindHostRecord(apiKey string, secretKey string, domainId int, subDomain string) int {
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
		if v.Host == subDomain {
			return v.Id
		}
	}

	return -1
}

func cloudxnsRequest(apiKey string, secretKey string, domain string, subDomain string, isInternal bool) error {
	// find the domain
	domainId := cloudxnsFindDomain(apiKey, secretKey, domain)
	if domainId == -1 {
		fmt.Println("can't find domain in list", domain)
		return errors.New("domain not exists")
	}
	// find the host
	hostRecordId := cloudxnsFindHostRecord(apiKey, secretKey, domainId, subDomain)
	if hostRecordId == -1 {
		fmt.Println("can't find host record in list", subDomain)
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

	newIP := currentExternalIPv4
	if isInternal {
		newIP = currentInternalIPv4
	}
	postValues := make(map[string]interface{})
	if foundRecord {
		// update
		postValues["domain_id"] = domainId
		postValues["host"] = subDomain
		postValues["value"] = newIP
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
			fmt.Printf("[%v] Updating CloudXNS resolve item failed: %v", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("A record updated to cloudXNS: %s.%s => %s\n", subDomain, domain, currentExternalIPv4)
	} else {
		// insert
		postValues["domain_id"] = fmt.Sprintf("%d", domainId)
		postValues["host"] = subDomain
		postValues["value"] = newIP
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
			fmt.Printf("[%v] inserting CloudXNS resolve item failed: %v", time.Now(), err)
			return err
		}
		defer resp.Body.Close()
		fmt.Printf("A record inserted to cloudXNS: %s.%s => %s\n", subDomain, domain, newIP)
	}
	return nil
}
