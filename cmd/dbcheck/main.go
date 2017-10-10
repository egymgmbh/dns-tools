// package main provides the dbcheck tool which checks zone data stored in a
// directory for common loading errors
package main

import (
	"flag"
	"log"

	"github.com/egymgmbh/dns-tools/config"
	"github.com/egymgmbh/dns-tools/rrdb"
)

func main() {
	exitOK := true
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

	for _, mz := range config.ManagedZones {
		_, err := db.Zone(mz.FQDN, mz.TTL)
		if err != nil {
			log.Printf("Managed zone %v: %v", mz.FQDN, err)
			exitOK = false
			continue
		}
	}
	if exitOK {
		log.Print("Looks good!")
	} else {
		log.Fatal("Errors found!")
	}
}
