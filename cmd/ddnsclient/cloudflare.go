package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func cloudflareRequest(user string, token string, domain string, subDomain string) error {
	// Construct a new API object
	api, err := cloudflare.New(token, user)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch user details on the account
	u, err := api.UserDetails()
	if err != nil {
		log.Fatal(err)
	}
	// Print user details
	fmt.Println("Cloudflare user information:", u)

	// Fetch the zone ID
	id, err := api.ZoneIDByName(domain) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		log.Fatal(err)
	}

	// Fetch zone details
	zone, err := api.ZoneDetails(id)
	if err != nil {
		log.Fatal(err)
	}
	// Print zone details
	fmt.Println("Cloudflare zone detail:", zone)

	// Fetch all records for a zone
	recs, err := api.DNSRecords(id, cloudflare.DNSRecord{Type: "A", Name: subDomain + "." + domain})
	if err != nil {
		log.Fatal(err)
	}

	r := cloudflare.DNSRecord{
		Type:    "A",
		Name:    subDomain + "." + domain,
		Content: currentExternalIPv4,
		ZoneID:  id,
	}
	if len(recs) == 0 {
		// insert a new record
		_, err = api.CreateDNSRecord(id, r)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("[%v] A record created to cloudflare: %s.%s => %s\n", time.Now(), subDomain, domain, currentExternalIPv4)
		}
	} else {
		// update
		err = api.UpdateDNSRecord(id, recs[0].ID, r)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("[%v] A record updated to cloudflare: %s.%s => %s\n", time.Now(), subDomain, domain, currentExternalIPv4)
		}
	}

	return nil
}
