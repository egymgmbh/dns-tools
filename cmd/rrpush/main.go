// package main provides the rrpush which pushes zone information to Cloud DNS
package main

import (
	"flag"
	"log"
	"time"

	"github.com/fatih/color"
	clouddns "google.golang.org/api/dns/v1"

	"github.com/egymgmbh/dns-tools/config"
	"github.com/egymgmbh/dns-tools/gcp"
	"github.com/egymgmbh/dns-tools/rrdb"
)

func main() {
	exitOK := true
	configFile := flag.String("config-file", "config.yml",
		"DNS Tools configuration file.")
	delay := flag.String("delay", "10s",
		"Safeguard: Wait [delay] before taking action on Cloud DNS.")
	dryRun := flag.Bool("dry-run", true,
		"Do not take action on Cloud DNS. Just pretend.")
	gcpSAFile := flag.String("gcp-sa-file", "secret/gcp-sa.json",
		"Google Cloud Platform Service Account file in JSON format.")
	noColor := flag.Bool("no-color", false, "Do not colorize output.")
	flag.Parse()

	// validate flags
	delayDuration, err := time.ParseDuration(*delay)
	if err != nil {
		log.Fatalf("parse delay: %v", err)
	}
	config, err := config.New(*configFile)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	service, projectID, err := gcp.GetDNSService(*gcpSAFile, *dryRun)
	if err != nil {
		log.Fatalf("get GCP service: %v", err)
	}
	color.NoColor = *noColor

	// load local data
	db, err := rrdb.NewFromDirectory(config.ZoneDataDirectory)
	if err != nil {
		log.Fatal(err)
	}

	// fetch current managed zones and make a hash map for faster access
	gcpMZListResponse, err := service.ManagedZones.List(projectID).Do()
	if err != nil {
		log.Fatalf("list managed zones: %v", err)
	}
	gcpManagedZones := make(map[string]string)
	for _, mz := range gcpMZListResponse.ManagedZones {
		gcpManagedZones[mz.DnsName] = mz.Name
	}

	// now we walk through the list of locally configured managed zones and
	// try to find them on Cloud DNS. If we find a zone, we will fetch the current
	// records and compare them with what our database wants to be there. We then
	// calculate a diff and log the change before we apply it.
	totalMissingInDatabase := 0
	totalMissingOnCloudDNS := 0
	totalFailed := 0
	totalDeletions := 0
	totalAdditions := 0
	for _, mz := range config.ManagedZones {
		log.SetPrefix(mz.FQDN + " ")
		// check zone's availability on Cloud DNS
		if _, ok := gcpManagedZones[mz.FQDN]; !ok {
			color.Set(color.FgHiYellow)
			log.Printf("Cloud DNS: zone not found")
			color.Unset()
			totalMissingOnCloudDNS++
			exitOK = false
			continue
		}

		// get zone's records from local database
		records, err := db.Zone(mz.FQDN, mz.TTL)
		if err != nil {
			color.Set(color.FgHiYellow)
			log.Printf("local database: %v", err)
			color.Unset()
			totalMissingInDatabase++
			exitOK = false
			continue
		}

		// get currently active records from Cloud DNS
		gcpRRListResponse, err := service.ResourceRecordSets.
			List(projectID, gcpManagedZones[mz.FQDN]).
			Do()
		if err != nil {
			color.Set(color.FgHiYellow)
			log.Printf("Cloud DNS: %v", err)
			color.Unset()
			totalFailed++
			exitOK = false
			continue
		}

		// We create a change request, which at this time, is very simple:
		// Delete all current records.
		// Add all wanted records.
		change := clouddns.Change{}
		change.Deletions = gcp.FilterRRSets(gcpRRListResponse.Rrsets, mz.FQDN)
		change.Additions = gcp.RRDBRecordsToCloudDNSRecords(records)
		// Usually, most of the records we want on Cloud DNS are already there from
		// a previous deployment. So we remove this duplicates from change.Deletions
		// and change.Additions. This leaves us with a nice diff and we only deploy
		// this diff.
		gcp.RemoveDuplicatesFromChange(&change)

		// Print the actual change (read: the diff) in a human-friendly way
		nDeletions := len(change.Deletions)
		nAdditions := len(change.Additions)
		if nDeletions == 0 && nAdditions == 0 {
			log.Println("nothing to change")
			continue
		} else {
			if nDeletions > 0 {
				log.Printf("%v records to be deleted", nDeletions)
				color.Set(color.FgRed)
				for _, line := range gcp.FormatRRSets(change.Deletions) {
					log.Print(line)
				}
				color.Unset()
			}
			if nAdditions > 0 {
				log.Printf("%v records to be added", nAdditions)
				color.Set(color.FgGreen)
				for _, line := range gcp.FormatRRSets(change.Additions) {
					log.Print(line)
				}
				color.Unset()
			}
		}

		// enforcing deployment delay
		if delayDuration > 0 {
			log.Printf("delaying change for %v seconds...", delayDuration)
			log.Printf("last chance to abort!")
			time.Sleep(delayDuration)
		}

		// back out if this is a dry-run
		if *dryRun {
			color.Set(color.FgHiYellow)
			log.Printf("skipping action! (dry run)")
			color.Unset()
			continue
		}

		// uploading change
		log.Printf("requesting change...")
		chg := &change
		chg, err = service.Changes.
			Create(projectID, gcpManagedZones[mz.FQDN], chg).
			Do()
		if err != nil {
			color.Set(color.FgHiYellow)
			log.Printf("request failed: %v", err)
			color.Unset()
			totalFailed++
			continue
		}
		pollCount := 0
		for chg.Status == "pending" && pollCount < 30 {
			log.Println("request pending...")
			time.Sleep(500 * time.Millisecond)
			pollCount++
			chg, err = service.Changes.
				Get(projectID, gcpManagedZones[mz.FQDN], chg.Id).
				Do()
			if err != nil {
				color.Set(color.FgHiYellow)
				log.Printf("request failed: %v", err)
				color.Unset()
				totalFailed++
				continue
			}
		}
		if chg != nil {
			log.Printf("request status: %v", chg.Status)
		}
		// update stats
		totalDeletions += nDeletions
		totalAdditions += nAdditions
	}
	log.SetPrefix("summary ")
	log.Printf("%v records removed, %v records created",
		totalDeletions, totalAdditions)
	log.Printf("%v managed zones, %v OK, %v failed, "+
		"%v missing in local database, %v missing on Cloud DNS",
		len(config.ManagedZones),
		len(config.ManagedZones)-totalFailed-totalMissingInDatabase-totalMissingOnCloudDNS,
		totalFailed,
		totalMissingInDatabase,
		totalMissingOnCloudDNS)
	if !exitOK {
		log.Fatal("some errors occurred")
	}
}
