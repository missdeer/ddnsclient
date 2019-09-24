package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/missdeer/ddnsclient/models"
)

var (
	dnspodDomainList = &models.DnspodDomainList{}
)

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
			"value":       {currentExternalIPv4},
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

		fmt.Printf("[%v] A record inserted into DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIPv4)
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
			"value":       {currentExternalIPv4},
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
		fmt.Printf("[%v] A record updated to DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIPv4)
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
			"value":          {currentExternalIPv4},
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

		fmt.Printf("[%v] A record inserted into DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIPv4)
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
			"value":          {currentExternalIPv4},
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
		fmt.Printf("[%v] A record updated to DNSPOD: %s.%s => %s\n", time.Now(), sub_domain, domain, currentExternalIPv4)
	}

	return nil
}
