package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

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
