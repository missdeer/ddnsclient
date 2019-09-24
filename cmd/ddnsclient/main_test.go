package main

import (
	"log"
	"testing"
)

func TestGetCurrentExternalIP(t *testing.T) {
	insecureSkipVerify = false
	ifconfigURL = "https://ifconfig.minidump.info"
	ip, err := getCurrentExternalIP(true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(ip)
	ip, err = getCurrentExternalIP(false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(ip)
}
