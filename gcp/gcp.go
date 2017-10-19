// Package gcp provides helper functions for connecting to the Cloud DNS API and
// for handling Cloud DNS-specific data types
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"

	"golang.org/x/oauth2/google"
	clouddns "google.golang.org/api/dns/v1"

	"github.com/egymgmbh/dns-tools/rrdb"
)

// GetDNSService creates a CloudDNS API service from a service account file
func GetDNSService(gcpSAFile string, readonly bool) (*clouddns.Service, string, error) {
	// read and parse Service Account file
	data, err := ioutil.ReadFile(gcpSAFile)
	if err != nil {
		return nil, "", fmt.Errorf("read Service Account file: %v", err)
	}

	var dataJSON struct {
		ProjectID string `json:"project_id"`
	}
	err = json.Unmarshal(data, &dataJSON)
	if err != nil {
		return nil, "", fmt.Errorf("parse Service Account file: %v", err)
	}
	projectID := dataJSON.ProjectID
	if projectID == "" {
		return nil, "",
			fmt.Errorf("parse Service Account file: project ID not found")
	}

	// define scope
	scope := clouddns.NdevClouddnsReadonlyScope
	if !readonly {
		scope = clouddns.NdevClouddnsReadwriteScope
	}
	// get config from web token
	conf, err := google.JWTConfigFromJSON(data, scope)
	if err != nil {
		return nil, "", fmt.Errorf("get Service Account config: %v", err)
	}

	// get CloudDNS API servive
	client := conf.Client(context.Background())
	service, err := clouddns.New(client)
	if err != nil {
		return nil, "", fmt.Errorf("create API service: %v", err)
	}

	return service, projectID, nil
}

// RRDBRecordsToCloudDNSRecords converts RRDB records to CloudDNS
// records (type: ResourceRecordSet)
func RRDBRecordsToCloudDNSRecords(in []*rrdb.Record) []*clouddns.ResourceRecordSet {
	var out []*clouddns.ResourceRecordSet
	for _, record := range in {
		out = append(out, &clouddns.ResourceRecordSet{
			Kind:    "dns#resourceRecordSet",
			Name:    record.FQDN,
			Type:    record.RType,
			Ttl:     int64(record.TTL),
			Rrdatas: record.RDatas,
		})
	}
	return out
}

func removeNilPointersFromRRS(in []*clouddns.ResourceRecordSet) []*clouddns.ResourceRecordSet {
	out := []*clouddns.ResourceRecordSet{}
	for _, item := range in {
		if item == nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

// recordID returns a unique ID for a ResourceRecordSet to be used as key in
// hash maps
func recordID(record *clouddns.ResourceRecordSet) string {
	return fmt.Sprintf("%v|%v|%v|%v",
		record.Kind, record.Name, record.Type, record.Ttl)
}

// RemoveDuplicatesFromChange compresses a CloudDNS change by removing
// deletions and additions that would cancel each other out
func RemoveDuplicatesFromChange(change *clouddns.Change) {
	// build a map of the deletions for faster access
	delIdxs := make(map[string]int)
	for idx, record := range change.Deletions {
		id := recordID(record)
		delIdxs[id] = idx
	}

	// iterate through additions and point all duplicate entries to nil
	// duplicate means, there is an addition that is identical to a deletion
	for idx, record := range change.Additions {
		id := recordID(record)
		delIdx, ok := delIdxs[id]
		if !ok ||
			change.Deletions[delIdx] == nil ||
			change.Additions[idx] == nil ||
			!reflect.DeepEqual(change.Deletions[delIdx].Rrdatas,
				change.Additions[idx].Rrdatas) {
			continue
		}
		change.Additions[idx] = nil
		change.Deletions[delIdx] = nil
	}
	change.Additions = removeNilPointersFromRRS(change.Additions)
	change.Deletions = removeNilPointersFromRRS(change.Deletions)
}

// FormatRRSets formats a resource record set in a human readable way and
// returns a slice of strings that can be used for printing to screen or log
func FormatRRSets(rrsets []*clouddns.ResourceRecordSet) []string {
	var out []string
	for _, rrset := range rrsets {
		out = append(out, fmt.Sprintf("*%v %v %v",
			rrset.Name, rrset.Type, rrset.Ttl))
		for _, rdata := range rrset.Rrdatas {
			out = append(out, fmt.Sprintf(" *%v", rdata))
		}
	}
	return out
}

// FilterRRSets removes resource record sets that must not be managed by
// dns-tools from a list. This is a safeguard.
func FilterRRSets(rrsets []*clouddns.ResourceRecordSet, fqdn string) []*clouddns.ResourceRecordSet {
	filtered := []*clouddns.ResourceRecordSet{}
	for _, rr := range rrsets {
		// ignore everything that is not a ResourceRecordSet
		if rr.Kind != "dns#resourceRecordSet" {
			continue
		}
		// don't even look at NS records for the zone itself
		if rr.Type == "NS" && rr.Name == fqdn {
			continue
		}
		// check everything else
		if rr.Type == "NS" || rr.Type == "MX" || rr.Type == "TXT" || rr.Type == "CNAME" ||
			rr.Type == "A" || rr.Type == "AAAA" {
			filtered = append(filtered, rr)
		}
	}
	return filtered
}
