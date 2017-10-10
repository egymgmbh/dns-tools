// Package config parses a YAML-formatted dns-tools configuration file for
// further use in the tools
package config

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/egymgmbh/dns-tools/lib"

	yaml "gopkg.in/yaml.v2"
)

var (
	regexHostname = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])\.$`)
)

// ManagedZoneDefaults holds the default configuration fo  managed zones
type ManagedZoneDefaults struct {
	TTL int
}

// ManagedZoneConfig holds a managed zone's configuration
type ManagedZoneConfig struct {
	FQDN string
	TTL  int
}

// Config holds the dns-tools configuration
type Config struct {
	ZoneDataDirectory string
	Defaults          ManagedZoneDefaults
	ManagedZones      []ManagedZoneConfig
}

// yamlConfig holds exactly one dns-tools configuration
type yamlConfig struct {
	Config Config
}

// New creates a new database instance
func New(fname string) (*Config, error) {
	yamlConfigData := yamlConfig{}
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	err = yaml.UnmarshalStrict(data, &yamlConfigData)
	if err != nil {
		return nil, err
	}
	config := yamlConfigData.Config

	// verify defaults
	err = checkTTL(config.Defaults.TTL)
	if err != nil {
		return nil, fmt.Errorf("defaults: %v", err)
	}

	// verify individual managed zones and set default TTL if no individual TTL
	// configured
	seen := make(map[string]bool)
	for idx := range config.ManagedZones {
		mz := &config.ManagedZones[idx] // uses actual data, not local copy
		// apply defaults to unset values
		if mz.TTL == 0 {
			mz.TTL = config.Defaults.TTL
		}
		// check name
		err = checkFQDN(mz.FQDN)
		if err != nil {
			return nil, fmt.Errorf("managed zone %v: %v", mz.FQDN, err)
		}
		// check managed zone default TTL
		err = checkTTL(mz.TTL)
		if err != nil {
			return nil, fmt.Errorf("managed zone %v: %v", mz.FQDN, err)
		}
		// check for duplicate zones
		if _, ok := seen[mz.FQDN]; ok {
			return nil, fmt.Errorf("managed zone %v: duplicate entry", mz.FQDN)
		}
		seen[mz.FQDN] = true
	}
	return &config, nil
}

func checkFQDN(fqdn string) error {
	if !regexHostname.MatchString(fqdn) {
		return fmt.Errorf("invalid FQDN: %v", fqdn)
	}
	return nil
}

func checkTTL(ttl int) error {
	err := lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// we can not allow zero TTLs in configuration files because a zero means
	// default TTL, and here in the configuration we set the default TTLs for the
	// zones
	if ttl == 0 {
		return fmt.Errorf("invalid TTL: %v", ttl)
	}
	return nil
}
