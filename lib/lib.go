// Package lib provides a library of commonly used functions for dns-tools
package lib

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	regexHostname = regexp.MustCompile(`^(([a-zA-Z0-9_]|[a-zA-Z0-9_][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])\.$`)
)

// Lookup looks up a resource record and returns the associated
// data as a string slice. It is very fast but not very precise, e.g.
// with CNAME records and on some systems with NS records, the results my vary.
// Critics may even find a way to blame systemd for this ;)
// related: https://github.com/systemd/systemd/issues/5897
func Lookup(fqdn, rtype string) ([]string, error) {
	var rdatas []string
	err := IsValidFQDN(fqdn)
	if err != nil {
		return rdatas, err
	}
	switch rtype {
	case "NS":
		var namservers []*net.NS
		namservers, err = net.LookupNS(fqdn)
		for _, namserver := range namservers {
			rdatas = append(rdatas, namserver.Host)
		}
	case "CNAME":
		var tmp string
		tmp, err = net.LookupCNAME(fqdn)
		rdatas = []string{tmp}
	case "MX":
		var mailservers []*net.MX
		mailservers, err = net.LookupMX(fqdn)
		for _, mailserver := range mailservers {
			rdatas = append(rdatas, fmt.Sprintf("%v %v", mailserver.Pref, mailserver.Host))
		}
	case "TXT":
		rdatas, err = net.LookupTXT(fqdn)
	case "AAAA":
		var addresses []string
		addresses, err = net.LookupHost(fqdn)
		for _, address := range addresses {
			if IsValidIPv6(address) == nil {
				rdatas = append(rdatas, address)
			}
		}
	case "A":
		var addresses []string
		addresses, err = net.LookupHost(fqdn)
		for _, address := range addresses {
			if IsValidIPv4(address) == nil {
				rdatas = append(rdatas, address)
			}
		}
	default:
		return rdatas, fmt.Errorf("unsupported record type: %v", rtype)
	}
	return rdatas, err
}

// RDatasEqual compares two unsorted slices of resource record data and
// returns true if their contents are the same
func RDatasEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, rdata := range a {
		seen[rdata]++
	}
	for _, rdata := range b {
		if _, ok := seen[rdata]; !ok {
			return false
		}
		seen[rdata]--
	}
	for _, x := range seen {
		if x != 0 {
			return false
		}
	}
	return true
}

// MakeFQDN creafts a fully qualified domain name from a name and a zone.
func MakeFQDN(name, zone string) string {
	if name == "@" {
		return zone
	}
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "." + zone
}

// IsValidIPv4 validates a string that contains an IPv4 address
func IsValidIPv4(address string) error {
	ip := net.ParseIP(address)
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address: %v", address)
	}
	if ip.To4() == nil {
		return fmt.Errorf("invalid IPv4 address: %v", address)
	}
	return nil
}

// IsValidIPv6 validates a string that contains an IPv6 address
func IsValidIPv6(address string) error {
	ip := net.ParseIP(address)
	if ip == nil {
		return fmt.Errorf("invalid IPv6 address: %v", address)
	}
	if !strings.Contains(ip.String(), ":") {
		return fmt.Errorf("invalid IPv6 address: %v", address)
	}
	return nil
}

// IsValidFQDN validates a string that contains a fully qualified domain
// name
func IsValidFQDN(fqdn string) error {
	if !regexHostname.MatchString(fqdn) {
		return fmt.Errorf("invalid FQDN: %v", fqdn)
	}
	return nil
}

// IsValidTTL validates an integer for an acceptable DNS Time To Live
// value
func IsValidTTL(ttl int) error {
	// max: https://tools.ietf.org/html/rfc2181#section-8
	if ttl < 0 || ttl > ((1<<31)-1) {
		return fmt.Errorf("invalid TTL: %v", ttl)
	}
	return nil
}

// TextToQuotedStrings prepares a text for usage in a TXT record where space separated
// strings are quoted
func TextToQuotedStrings(s string) string {
	s = strings.TrimSpace(s)
	ss := strings.Split(s, " ")
	if len(ss) == 1 && ss[0] == "" {
		return ""
	}
	s = fmt.Sprintf("%q", ss)
	s = s[1 : len(s)-1]
	return s
}
