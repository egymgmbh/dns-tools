package rrdb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egymgmbh/dns-tools/lib"
)

// RRDB holds a resource record database
type RRDB struct {
	root node
	// wrapped in a struct to leave room for future features that require fields
}

// This is a trie node. Tries are beautiful!
// https://en.wikipedia.org/wiki/Trie
type node struct {
	fqdn       string
	children   map[string]*node
	parent     *node
	nsRDatas   []string
	nsTTL      int
	mxRDatas   []string
	mxTTL      int
	txtRDatas  []string
	txtTTL     int
	txtSPF1    bool
	txtDKIM1   bool
	cnameRdata string
	cnameTTL   int
	aRDatas    []string
	aTTL       int
	aaaaRDatas []string
	aaaaTTL    int
}

// Record holds a DNS resource record of a particular type for a FQDN
type Record struct {
	FQDN   string
	RType  string
	TTL    int
	RDatas []string
}

// Mailserver holds the preference and hostname of a single mailserver's entry
type Mailserver struct {
	Preference uint16
	Hostname   string
}

// New creates a new label database instance
func New() *RRDB {
	return &RRDB{
		root: node{
			children: make(map[string]*node),
			parent:   nil,
		},
	}
}

// Records retrieves all records of a FQDN
func (db *RRDB) Records(fqdn string, ttl int) ([]*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.records(ttl, false), nil
}

// Zone retrieves all records of a FQDN and all records of all its children
func (db *RRDB) Zone(fqdn string, ttl int) ([]*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.records(ttl, true), nil
}

func (nd *node) records(ttl int, withChildren bool) []*Record {
	records := []*Record{}

	record, err := nd.ns(ttl)
	if err == nil {
		records = append(records, record)
	}

	record, err = nd.mx(ttl)
	if err == nil {
		records = append(records, record)
	}

	record, err = nd.txt(ttl)
	if err == nil {
		records = append(records, record)
	}

	record, err = nd.cname(ttl)
	if err == nil {
		records = append(records, record)
	}

	record, err = nd.a(ttl)
	if err == nil {
		records = append(records, record)
	}

	record, err = nd.aaaa(ttl)
	if err == nil {
		records = append(records, record)
	}

	if withChildren {
		for _, next := range nd.children {
			records = append(records, next.records(ttl, true)...)
		}
	}
	return records
}

/* --- helper functions ----------------------------------------------------- */

func (nd *node) hasRecords() bool {
	return nd.hasNS() || nd.hasMX() || nd.hasTXT() ||
		nd.hasCNAME() || nd.hasA() || nd.hasAAAA()
}

func (nd *node) hasNS() bool {
	return len(nd.nsRDatas) != 0
}

func (nd *node) hasMX() bool {
	return len(nd.mxRDatas) != 0
}

func (nd *node) hasTXT() bool {
	return len(nd.txtRDatas) != 0
}

func (nd *node) hasCNAME() bool {
	return len(nd.cnameRdata) != 0
}

func (nd *node) hasA() bool {
	return len(nd.aRDatas) != 0
}

func (nd *node) hasAAAA() bool {
	return len(nd.aaaaRDatas) != 0
}

func (nd *node) hasChildren() bool {
	return len(nd.children) != 0
}

// node finds a node identified by a set of hierachical labels (ls means label
// set). This function is recursive and calls itself usually a couple of times
// reducing the applied label set in every recursion by decreasing the index.
// Example: FQDN foo.example.com. has label set ['foo', 'example', 'com']
// and is recursed as first 'com' then 'example' and finally 'foo'.
func (nd *node) node(ls []string, idx int, create bool) (*node, error) {
	// Hooray, it's us!
	if idx < 0 {
		return nd, nil
	}
	// we can not go deeper if we have a NS record. Our authority ends there :/
	if nd.hasNS() {
		return nil, fmt.Errorf("FQDN outside authority")
	}
	// now we can go deeper
	label := ls[idx]
	if _, ok := nd.children[label]; !ok {
		// Ah, snap, no node for that label!
		if create {
			// Let's summon one :)
			nd.children[label] = &node{
				fqdn:     strings.Join(ls[idx:], ".") + ".",
				children: make(map[string]*node),
				parent:   nd,
			}
		} else {
			return nil, fmt.Errorf("FQDN not found")
		}
	}
	// Nothing to see here, digging deeper.
	return nd.children[label].node(ls, idx-1, create)
}

func (db *RRDB) node(fqdn string, create bool) (*node, error) {
	err := lib.IsValidFQDN(fqdn)
	if err != nil {
		return nil, err
	}
	ls := strings.Split(strings.Trim(fqdn, "."), ".")
	return db.root.node(ls, len(ls)-1, create)
}

/* --- NS ------------------------------------------------------------------- */

// SetNS sets the NS records of a FQDN
func (db *RRDB) SetNS(fqdn string, ttl int, rdatas []string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// check empty
	if len(rdatas) == 0 {
		return fmt.Errorf("rdatas: empty")
	}
	if nd.hasNS() {
		return fmt.Errorf("NS record already set")
	}
	// validation and duplicate detection
	seen := make(map[string]bool)
	for _, rdata := range rdatas {
		err = lib.IsValidFQDN(rdata)
		if err != nil {
			return fmt.Errorf("rdata: %v", err)
		}
		if _, ok := seen[rdata]; ok {
			return fmt.Errorf("rdata: duplicate entry: %v", rdata)
		}
		seen[rdata] = true
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	// A FQDN can not have any other records it holds a delegation
	if nd.hasRecords() {
		return fmt.Errorf("conflicting records")
	}
	// A FQDN can not have sub-labels (children) when it holds a delegation
	if nd.hasChildren() {
		return fmt.Errorf("cannot delegate FQDN with children")
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.nsTTL = ttl
	nd.nsRDatas = rdatas
	return nil
}

// NS retrieves the NS record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) NS(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.ns(ttl)
}

func (nd *node) ns(ttl int) (*Record, error) {
	if !nd.hasNS() {
		return nil, fmt.Errorf("FQDN has no NS record")
	}
	if nd.nsTTL != 0 {
		ttl = nd.nsTTL
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "NS",
		TTL:    ttl,
		RDatas: nd.nsRDatas,
	}, nil
}

/* --- MX ------------------------------------------------------------------- */

// SetMX adds a MX record to a FQDN
func (db *RRDB) SetMX(fqdn string, ttl int, rdatas []string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// check empty
	if len(rdatas) == 0 {
		return fmt.Errorf("rdatas: empty")
	}
	if nd.hasMX() {
		return fmt.Errorf("MX record already set")
	}
	// accept RFC7505 null MX and skip tests in that case
	if !(len(rdatas) == 1 && rdatas[0] == "0 .") {
		// validation and duplicate detection
		seen := make(map[string]bool)
		for _, rdata := range rdatas {
			mxLine := strings.SplitN(rdata, " ", 2)
			preference, err := strconv.ParseInt(mxLine[0], 10, 64)
			if err != nil {
				return fmt.Errorf("rdata: invalid preference: %v", mxLine[0])
			}
			hostname := mxLine[1]
			// check preference
			if preference < 0 || preference > 65535 {
				return fmt.Errorf("rdata: invalid preference: %v", preference)
			}
			// check hostname
			err = lib.IsValidFQDN(hostname)
			if err != nil {
				return fmt.Errorf("rdata: %v", err)
			}
			if _, ok := seen[hostname]; ok {
				return fmt.Errorf("rdata: duplicate entry: %v", rdata)
			}
			seen[hostname] = true
		}
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	// A FQDN can not have a MX record when it also has a NS or CNAME record
	if nd.hasNS() || nd.hasCNAME() {
		return fmt.Errorf("conflicting record")
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.mxTTL = ttl
	nd.mxRDatas = rdatas
	return nil
}

// MX retrieves the MX record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) MX(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.mx(ttl)
}

func (nd *node) mx(ttl int) (*Record, error) {
	if !nd.hasMX() {
		return nil, fmt.Errorf("FQDN has no MX record")
	}
	if nd.mxTTL != 0 {
		ttl = nd.mxTTL
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "MX",
		TTL:    ttl,
		RDatas: nd.mxRDatas,
	}, nil
}

/* --- TXT ------------------------------------------------------------------ */

// AddTXT adds a TXT record to a FQDN
func (db *RRDB) AddTXT(fqdn string, ttl int, rdata string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// do not allow to reset previously set TTL
	if nd.txtTTL != 0 && ttl != nd.txtTTL {
		return fmt.Errorf("TTL already set")
	}
	// check empty
	if len(rdata) == 0 {
		return fmt.Errorf("rdata: empty")
	}
	// check maxlength
	if len(rdata) > 255 {
		return fmt.Errorf("rdata: too large")
	}
	// duplicate detection
	for _, txt := range nd.txtRDatas {
		if rdata == txt {
			return fmt.Errorf("rdata: duplicate entry: %v", rdata)
		}
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	// A FQDN can not have a TXT record when it also has a NS or CNAME record
	if nd.hasNS() || nd.hasCNAME() {
		return fmt.Errorf("conflicting records")
	}

	// some sanity checks to prevent the most common mistakes
	rdataLower := strings.ToLower(rdata)
	if strings.HasPrefix(rdataLower, "v=spf1 ") ||
		strings.HasPrefix(rdataLower, "v=spf1;") {
		if nd.txtSPF1 {
			return fmt.Errorf("rdata: SPF already set")
		}
		nd.txtSPF1 = true
	}
	if strings.HasPrefix(rdataLower, "v=dkim1 ") ||
		strings.HasPrefix(rdataLower, "v=dkim1;") {
		if nd.txtDKIM1 {
			return fmt.Errorf("rdata: DKIM already set")
		}
		nd.txtDKIM1 = true
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.txtTTL = ttl
	nd.txtRDatas = append(nd.txtRDatas, rdata)
	return nil
}

// TXT retrieves the TXT record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) TXT(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.txt(ttl)
}

func (nd *node) txt(ttl int) (*Record, error) {
	if !nd.hasTXT() {
		return nil, fmt.Errorf("FQDN has no TXT record")
	}
	if nd.txtTTL != 0 {
		ttl = nd.txtTTL
	}
	rdatas := []string{}
	for _, rdata := range nd.txtRDatas {
		rdatas = append(rdatas, fmt.Sprintf("%q", rdata))
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "TXT",
		TTL:    ttl,
		RDatas: rdatas,
	}, nil
}

/* --- CNAME ---------------------------------------------------------------- */

// SetCNAME sets the CNAME record of a FQDN
func (db *RRDB) SetCNAME(fqdn string, ttl int, rdata string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	if nd.hasCNAME() {
		return fmt.Errorf("CNAME record already set")
	}
	// validation
	err = lib.IsValidFQDN(rdata)
	if err != nil {
		return fmt.Errorf("rdata: %v", err)
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	/*
	 * RFC1912: A CNAME record is not allowed to coexist with any other data.
	 * RFC1034: If a CNAME RR is present at a node, no other data should be
	 * present; this ensures that the data for a canonical name and its aliases
	 * cannot be different.
	 */
	if nd.hasRecords() {
		return fmt.Errorf("conflicting records")
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.cnameTTL = ttl
	nd.cnameRdata = rdata
	return nil
}

// CNAME retrieves the CNAME record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) CNAME(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.cname(ttl)
}

func (nd *node) cname(ttl int) (*Record, error) {
	if !nd.hasCNAME() {
		return nil, fmt.Errorf("FQDN has no CNAME record")
	}
	if nd.cnameTTL != 0 {
		ttl = nd.cnameTTL
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "CNAME",
		TTL:    ttl,
		RDatas: []string{nd.cnameRdata},
	}, nil
}

/* --- A -------------------------------------------------------------------- */

// SetA adds an A record to a FQDN
func (db *RRDB) SetA(fqdn string, ttl int, rdatas []string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// check empty
	if len(rdatas) == 0 {
		return fmt.Errorf("rdatas: empty")
	}
	if nd.hasA() {
		return fmt.Errorf("A record already set")
	}
	// validation and duplicate detection
	seen := make(map[string]bool)
	for _, rdata := range rdatas {
		err = lib.IsValidIPv4(rdata)
		if err != nil {
			return fmt.Errorf("rdata: %v", err)
		}
		if _, ok := seen[rdata]; ok {
			return fmt.Errorf("rdata: duplicate entry: %v", rdata)
		}
		seen[rdata] = true
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	// A FQDN can not have an A record when it also has a NS or CNAME record
	if nd.hasNS() || nd.hasCNAME() {
		return fmt.Errorf("conflicting records")
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.aTTL = ttl
	nd.aRDatas = rdatas
	return nil
}

// A retrieves the A record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) A(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.a(ttl)
}

func (nd *node) a(ttl int) (*Record, error) {
	if !nd.hasA() {
		return nil, fmt.Errorf("FQDN has no A record")
	}
	if nd.aTTL != 0 {
		ttl = nd.aTTL
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "A",
		TTL:    ttl,
		RDatas: nd.aRDatas,
	}, nil
}

/* --- AAAA ----------------------------------------------------------------- */

// SetAAAA adds an AAAA record to a FQDN
func (db *RRDB) SetAAAA(fqdn string, ttl int, rdatas []string) error {
	nd, err := db.node(fqdn, true)
	if err != nil {
		return err
	}
	err = lib.IsValidTTL(ttl)
	if err != nil {
		return err
	}
	// check empty
	if len(rdatas) == 0 {
		return fmt.Errorf("rdatas: empty")
	}
	if nd.hasAAAA() {
		return fmt.Errorf("AAAA record already set")
	}
	// validation and duplicate detection
	seen := make(map[string]bool)
	for _, rdata := range rdatas {
		err = lib.IsValidIPv6(rdata)
		if err != nil {
			return fmt.Errorf("rdata: %v", err)
		}
		if _, ok := seen[rdata]; ok {
			return fmt.Errorf("rdata: duplicate entry: %v", rdata)
		}
		seen[rdata] = true
	}

	/* --- BEGIN: logic checks ---------------------------------------------- */
	// A FQDN can not have an AAAA record when it also has a NS or CNAME record
	if nd.hasNS() || nd.hasCNAME() {
		return fmt.Errorf("conflicting records")
	}
	/* --- END: logic checks ------------------------------------------------ */

	// all good
	nd.aaaaTTL = ttl
	nd.aaaaRDatas = rdatas
	return nil
}

// AAAA retrieves the AAAA record of a FQDN. If the record has no individual
// TTL, a default TTL (paramter ttl) will be inserted.
func (db *RRDB) AAAA(fqdn string, ttl int) (*Record, error) {
	nd, err := db.node(fqdn, false)
	if err != nil {
		return nil, err
	}
	return nd.aaaa(ttl)
}

func (nd *node) aaaa(ttl int) (*Record, error) {
	if !nd.hasAAAA() {
		return nil, fmt.Errorf("FQDN has no AAAA record")
	}
	if nd.aaaaTTL != 0 {
		ttl = nd.aaaaTTL
	}
	return &Record{
		FQDN:   nd.fqdn,
		RType:  "AAAA",
		TTL:    ttl,
		RDatas: nd.aaaaRDatas,
	}, nil
}
