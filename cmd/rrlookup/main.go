// package main provides the rrlookup tool which verifies DNS records
// by comparing lookup results with values loaded from a configuration
package main

import (
	"flag"
	"log"

	"github.com/egymgmbh/dns-tools/config"
	"github.com/egymgmbh/dns-tools/lib"
	"github.com/egymgmbh/dns-tools/rrdb"
)

func main() {
	configFile := flag.String("config-file", "config.yml",
		"DNS Tools configuration file.")
	flag.Parse()

	config, err := config.New(*configFile)
	if err != nil {
		log.Fatalf("get configuration: %v", err)
	}

	db, err := rrdb.NewFromDirectory(config.ZoneDataDirectory)
	if err != nil {
		log.Fatal(err)
	}
	totalNotInDatabase := 0
	totalResolverError := 0
	totalMismatch := 0
	totalOK := 0

	for _, mz := range config.ManagedZones {
		records, err := db.Zone(mz.FQDN, 0)
		if err != nil {
			log.Printf("%v: %v", mz.FQDN, err)
			totalNotInDatabase++
			continue
		}
		for _, record := range records {
			actual, err := lib.Lookup(record.FQDN, record.RType)
			if err != nil {
				log.Printf("%v: resolver error", record.FQDN)
				totalResolverError++
				continue
			}
			if lib.RDatasEqual(actual, record.RDatas) {
				totalOK++
			} else {
				log.Printf("%v: want %v: %q", record.FQDN, record.RType, record.RDatas)
				log.Printf("%v: have %v: %q", record.FQDN, record.RType, actual)
				totalMismatch++
			}
		}
	}
	log.Printf("%v ok, %v mismatch, %v resolver errors, %v not in database",
		totalOK, totalMismatch, totalResolverError, totalNotInDatabase)
}
