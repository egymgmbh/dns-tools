// Package main provides the mzcreate tool that creates managed zones on
// Cloud DNS that are in the configuration file but missing on Cloud DNS.
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egymgmbh/dns-tools/config"
	"github.com/egymgmbh/dns-tools/gcp"
	"github.com/fatih/color"
	clouddns "google.golang.org/api/dns/v1"
)

// dnsNameToMZName converts a DNS zone name into a format that is
// accepted as managed zone name
func dnsNameToMZName(dnsName string) string {
	dnsName = strings.Trim(dnsName, ".")
	labels := strings.Split(dnsName, ".")
	lastLabel := len(labels) - 1
	for i := 0; i < len(labels)/2; i++ {
		labels[i], labels[lastLabel-i] = labels[lastLabel-i], labels[i]
	}
	return strings.Join(labels, "--")
}

func main() {
	exitOK := true
	configFile := flag.String("config-file", "config.yml",
		"DNS Tools configuration file.")
	dryRun := flag.Bool("dry-run", true,
		"Do not take action on Cloud DNS. Just pretend.")
	noColor := flag.Bool("no-color", false, "Do not colorize output.")
	gcpSAFile := flag.String("gcp-sa-file", "secret/gcp-sa.json",
		"Google Cloud Platform Service Account file in JSON format.")
	flag.Parse()

	color.NoColor = *noColor

	config, err := config.New(*configFile)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	service, projectID, err := gcp.GetDNSService(*gcpSAFile, *dryRun)
	if err != nil {
		log.Fatalf("get DNS API service: %v", err)
	}

	// fetch current managed zones and make a hash map for quick access
	mzlist, err := service.ManagedZones.List(projectID).Do()
	if err != nil {
		log.Fatalf("list managed zones: %v", err)
	}
	gcpManagedZones := make(map[string]bool)
	for _, mz := range mzlist.ManagedZones {
		gcpManagedZones[mz.DnsName] = true
	}

	// compare configured managed zones with Cloud DNS managed zones
	totalCreated := 0
	for _, mz := range config.ManagedZones {
		log.SetPrefix(mz.FQDN + " ")
		if _, ok := gcpManagedZones[mz.FQDN]; ok {
			log.Printf("OK")
			continue
		}
		color.Set(color.FgHiYellow)
		log.Printf("not on Cloud DNS")
		color.Unset()
		if *dryRun {
			color.Set(color.FgHiYellow)
			log.Printf("skipping action! (dry run)")
			color.Unset()
			continue
		}
		wantMZptr := &clouddns.ManagedZone{
			DnsName:     mz.FQDN,
			Name:        dnsNameToMZName(mz.FQDN),
			Description: fmt.Sprintf("created by mzcreate %v", time.Now()),
		}
		newMZ, err := service.ManagedZones.Create(projectID, wantMZptr).Do()
		if err != nil {
			color.Set(color.FgHiYellow)
			log.Printf("api call: %v", err)
			color.Unset()
			exitOK = false
			continue
		}
		log.Printf("created")
		log.Printf("name:        %v", newMZ.Name)
		log.Printf("nameservers: %q", newMZ.NameServers)
		totalCreated++
	}
	log.SetPrefix("summary")
	log.Printf("%v managed zone create", totalCreated)
	if !exitOK {
		log.Fatal("some errors occurred")
	}
}
