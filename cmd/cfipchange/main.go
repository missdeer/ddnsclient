package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cloudflare/cloudflare-go"
)

var (
	cmd               string
	cfKey             string
	cfEmail           string
	recordName        string
	recordContent     string
	recordType        string
	recordContentFrom string
	recordContentTo   string
)

func listRecord(api *cloudflare.API) {
	zones, err := api.ListZones()
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range zones {
		records, err := api.DNSRecords(zone.ID, cloudflare.DNSRecord{})
		if err != nil {
			log.Println("getting dns records failed for ", zone.Host, err)
			continue
		}
		for _, record := range records {
			if record.Type == "A" {
				fmt.Printf("record %s => %s\n", record.Name, record.Content)
			}
		}
	}
}

func modifyRecord(api *cloudflare.API, rrType string, name string, content string) {
	zones, err := api.ListZones()
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range zones {
		records, err := api.DNSRecords(zone.ID, cloudflare.DNSRecord{})
		if err != nil {
			log.Println("getting dns records failed for ", zone.Host, err)
			continue
		}
		for _, record := range records {
			if record.Name == name && (rrType == "" || record.Type == rrType) {
				old := record.Content
				record.Content = content
				if err = api.UpdateDNSRecord(zone.ID, record.ID, record); err != nil {
					log.Println("update dns record failed", err)
				} else {
					log.Printf("dns record %s.%s updated from %s to %s\n", record.Name, zone.Host, old, content)
				}
				return
			}
		}
	}
}

func changeSpecifiedRecords(api *cloudflare.API, rrType string, from string, to string) {
	zones, err := api.ListZones()
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range zones {
		records, err := api.DNSRecords(zone.ID, cloudflare.DNSRecord{})
		if err != nil {
			log.Println("getting dns records failed for ", zone.Host, err)
			continue
		}
		for _, record := range records {
			if record.Content == from && (rrType == "" || record.Type == rrType) {
				record.Content = to
				if err = api.UpdateDNSRecord(zone.ID, record.ID, record); err != nil {
					log.Println("update dns record failed", err)
				} else {
					log.Printf("dns record %s.%s updated from %s to %s\n", record.Name, zone.Host, from, to)
				}
			}
		}
	}
}

func main() {
	cfKey = os.Getenv("CF_API_KEY")
	cfEmail = os.Getenv("CF_API_EMAIL")

	flag.StringVar(&cfKey, "key", cfKey, "Cloudflare API key")
	flag.StringVar(&cfEmail, "email", cfEmail, "Cloudflare account email")
	flag.StringVar(&cmd, "cmd", "list", "command: list, modify, change")
	flag.StringVar(&recordName, "name", "", "record name to be modified")
	flag.StringVar(&recordContent, "content", "", "record content modified to")
	flag.StringVar(&recordType, "type", "A", "record type to be modified or changed")
	flag.StringVar(&recordContentFrom, "from", "", "record content to be changed from")
	flag.StringVar(&recordContentTo, "to", "", "record content to be changed to")
	flag.Parse()

	api, err := cloudflare.New(cfKey, cfEmail)
	if err != nil {
		log.Fatal(err)
	}

	switch cmd {
	case "list":
		listRecord(api)
	case "modify":
		modifyRecord(api, recordType, recordName, recordContent)
	case "change":
		changeSpecifiedRecords(api, recordType, recordContentFrom, recordContentTo)
	}
}
