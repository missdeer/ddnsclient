package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dfordsoft/ddnsclient/models"
)

var (
	token      string
	user       string
	domain     string
	suffix     string
	prefixList string
	maxCount   int
)

func cloudflareCNAME(subDomain string) error {
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
		fmt.Println("request cloudflare all records failed.", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("reading cloudflare all records failed.", err)
		return err
	}

	recordList := new(models.CloudflareRecordList)
	if err = json.Unmarshal(body, &recordList); err != nil {
		fmt.Printf("unmarshalling cloudflare all records %s failed, %v\n", string(body), err)
		return err
	}

	// insert or update
	foundRecord := false
	var recordId string
	for _, v := range recordList.Response.Recs.Objs {
		if v.Type == "CNAME" && v.DisplayName == subDomain {
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
			"type":    {"CNAME"},
			"name":    {subDomain},
			"content": {fmt.Sprintf("%s.%s", subDomain, suffix)},
		})
		if err != nil {
			fmt.Println("request cloudflare new record failed.", err)
			return err
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("reading cloudflare new record failed.", err)
			return err
		}
		// extract the new record id
		respBody := new(models.CloudflareNewRecordResponseBody)
		if err = json.Unmarshal(body, respBody); err != nil {
			fmt.Println("unmarshalling cloudflare new record response body failed.", err)
			return err
		}
		recordId = respBody.Response.Rec.Obj.Id
		fmt.Printf("[%v] CNAME record inserted into cloudflare: %s.%s => %s.%s\n", time.Now(), subDomain, domain, subDomain, suffix)
		return nil
	}
	// update the record
	resp, err = client.PostForm(cloudflareAPIUrl, url.Values{
		"a":            {"rec_edit"},
		"tkn":          {token},
		"email":        {user},
		"z":            {domain},
		"type":         {"CNAME"},
		"service_mode": {"0"},
		"ttl":          {"1"},
		"id":           {recordId},
		"name":         {subDomain},
		"content":      {fmt.Sprintf("%s.%s", subDomain, suffix)},
	})
	if err != nil {
		fmt.Println("request cloudflare records edit failed", err)
		return err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("reading cloudflare record edit response failed.", err)
		return err
	}
	fmt.Printf("[%v] CNAME record update into cloudflare: %s.%s => %s.%s\n", time.Now(), subDomain, domain, subDomain, suffix)
	return nil
}

func main() {
	flag.StringVar(&domain, "domain", "", "your domain, such as xxx.com")
	flag.StringVar(&suffix, "suffix", "", "target domain, such as zzz.moe")
	flag.StringVar(&token, "token", "", "your cloudflare token")
	flag.StringVar(&user, "user", "", "your cloudflare user account")
	flag.StringVar(&prefixList, "prefix", "cn,kr,eu,tw,us,sg,jp,ru,hk", "prefix list")
	flag.IntVar(&maxCount, "max", 9, "max count")
	flag.Parse()

	if len(domain) == 0 || len(suffix) == 0 || len(token) == 0 || len(user) == 0 {
		flag.Usage()
		return
	}

	prefixes := strings.Split(prefixList, ",")
	for _, prefix := range prefixes {
		for i := 1; i <= maxCount; i++ {
			subDomain := fmt.Sprintf("%s-%d", prefix, i)
			cloudflareCNAME(subDomain)
		}
	}
	fmt.Println("Done!")
}
