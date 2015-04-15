package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

func getCurrentExternalIP() string {
	return ""
}

func basicAuthorizeHttpRequest(user string, password string, requestUrl string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.SetBasicAuth(user, password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Errorf("request %s failed", requestUrl)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func dnspodRequest(user string, password string, domain string, sub_domain string) {
	// get domainn id first
	// if the sub domain doesn't exist, add one
	// otherwise just update it
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

	for _, v := range setting.BasicAuthItems {
		basicAuthorizeHttpRequest(v.UserName, v.Password, v.Url)
	}
}
