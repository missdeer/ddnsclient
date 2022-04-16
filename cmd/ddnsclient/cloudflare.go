package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func cloudflareRequest(user string, token string, domain string, subDomain string, isInternal bool) error {
	// Construct a new API object
	api, err := cloudflare.New(token, user)
	if err != nil {
		log.Fatal(err)
		return err
	}

	ctx := context.Background()
	// Fetch user details on the account
	u, err := api.UserDetails(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}
	// Print user details
	fmt.Println("Cloudflare user information:", u)

	// Fetch the zone ID
	id, err := api.ZoneIDByName(domain) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Fetch zone details
	zone, err := api.ZoneDetails(ctx, id)
	if err != nil {
		log.Fatal(err)
		return err
	}
	// Print zone details
	fmt.Println("Cloudflare zone detail:", zone)

	// Fetch all records for a zone
	recs, err := api.DNSRecords(ctx, id, cloudflare.DNSRecord{Type: "A", Name: subDomain + "." + domain})
	if err != nil {
		log.Fatal(err)
		return err
	}

	newIP := currentExternalIPv4
	if isInternal {
		newIP = currentInternalIPv4
	}
	r := cloudflare.DNSRecord{
		Type:    "A",
		Name:    subDomain + "." + domain,
		Content: newIP,
		ZoneID:  id,
	}
	if len(recs) == 0 {
		// insert a new record
		_, err = api.CreateDNSRecord(ctx, id, r)
		if err != nil {
			fmt.Println(err)
			return err
		} else {
			fmt.Printf("[%v] A record created to cloudflare: %s.%s => %s\n", time.Now(), subDomain, domain, newIP)
		}
	} else {
		// update
		err = api.UpdateDNSRecord(ctx, id, recs[0].ID, r)
		if err != nil {
			fmt.Println(err)
			return err
		} else {
			fmt.Printf("[%v] A record updated to cloudflare: %s.%s => %s\n", time.Now(), subDomain, domain, newIP)
		}
	}

	return nil
}
