package rrdb

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testFQDN     = "foo.test."
	invalidFQDNs = []string{
		"",
		".",
		"example.com",
		"-foo.example.com",
		"-foo.example.com.",
		"--foo.example.com.",
		"foo-.example.com.",
		"foo--.example.com.",
	}
	validFQDNs = []string{
		"com.",
		"example.com.",
		"foo.example.com.",
		"f-o-o.example.com.",
		"xn--foo-bar.example.com.",
	}
	invalidTTLs   = []int{-300, -3, -2, -1, 2147483648, 2147483649, 2147483650}
	validTTLs     = []int{0, 1, 2, 3, 10, 300, 2147483645, 2147483646, 2147483647}
	validTTL      = 300
	otherValidTTL = 600
	validNSs      = [][]string{
		{
			"ns1.example.com.",
			"ns2.example.com.",
		},
		{
			"ns1.example.com.",
		},
	}
	validNS    = validNSs[0]
	invalidNSs = [][]string{
		{}, // empty
		{
			"ns1-.example.com.", // bad fqdn
		},
		{
			"ns1.example.com", // bad fqdns
			"ns2.example.com",
		}, {
			"ns1.example.com.",
			"ns1.example.com.", // duplicate
		}, {
			"ns1.example.com.",
			"ns1.example.com", // looks like duplicate, but misses root label
		},
	}
	invalidMXs = [][]string{
		{}, // empty
		{
			"10 mx.example.com.",
			"20 mx.example.com.", //duplicate hostname
		},
		{
			"10 mx.example.com.",
			"10 mx.example.com.", // full duplicate
		},
		{
			"100 mx1.example.com.",
			"900 mx2.example.com", // note the missing dot!
		},
		{
			"30 mx1.example-.com.", // invalid hostname
			"40 mx2.example.com.",
		},
		{
			"-1 mx1.example.com.", // invalid preference
			"40 mx2.example.com.",
		},
		{
			"66000 mx1.example.com.", // invalid preference
			"40 mx2.example.com.",
		},
		{
			"foo mx1.example.com.", // invalid preference
			"40 mx2.example.com.",
		},
	}
	validMXs = [][]string{
		{
			"10 mx1.example.com.",
			"20 mx2.example.com.",
		},
		{
			"0 mx.example.com.",
		},
		{
			"10 aspmx.l.google.com.",
			"20 alt1.aspmx.l.google.com.",
			"20 alt2.aspmx.l.google.com.",
			"30 aspmx2.googlemail.com.",
			"30 aspmx3.googlemail.com.",
			"30 aspmx4.googlemail.com.",
			"30 aspmx5.googlemail.com.",
		},
	}
	validMX     = []string{"10 mx.example.com."}
	invalidTXTs = [][]string{
		{""}, // empty string
		{
			"All watched over by machines of love and grace",
			"All watched over by machines of love and grace", // duplicate
		},
		{
			"v=spf1 include:bar.example.com ?all",
			"v=spf1 mx -all", // second SPF
		},
		{
			"v=DKIM1; t=100200; p=123456",
			"v=DKIM1 t=s p=123456", // second DKIM
		},
		{
			"Wie oft habe ich daran gedacht, wie ungleich die Glücksgüter in " +
				"unserem Leben verteilt sind! Warum hat das Schicksal Ihnen so " +
				"reizende Kinder gegeben, mit Ausnahme von Anatol, Ihrem Jüngsten, " +
				"den ich nicht liebe, fügte sie mit der Bestimmtheit eines " +
				"unerbittlichen Urteils hinzu, indem sie die Augenbrauen in die Höhe " +
				"zog. Sie wissen Ihr Glück nicht zu schätzen, also verdienen Sie es " +
				"auch nicht.", // too long
		},
	}
	validTXTs = []struct {
		in  []string
		out []string
	}{
		{
			in: []string{
				"Free drinks at the Foo bar!",
				"So long and thanks for all the fish...",
			},
			out: []string{
				"\"Free drinks at the Foo bar!\"",
				"\"So long and thanks for all the fish...\"",
			},
		},
		{
			in:  []string{"v=spf1 include:bar.example.com ?all"},
			out: []string{"\"v=spf1 include:bar.example.com ?all\""},
		},
		{
			in:  []string{"All watched over by machines of love and grace"},
			out: []string{"\"All watched over by machines of love and grace\""},
		},
		{
			in:  []string{"\"Atlas shrugged\", he replied."},
			out: []string{"\"\\\"Atlas shrugged\\\", he replied.\""},
		},
	}
	validTXT      = validTXTs[0]
	invalidCNAMEs = invalidFQDNs
	validCNAMEs   = validFQDNs
	validCNAME    = validCNAMEs[0]
	validCNAMEStr = []string{validCNAME}
	invalidAs     = [][]string{
		{}, // empty
		{
			"1.2.3",
			"192.0.2.256",
		},
		{
			"192.0.2.1",
			"192.0.2.1", // duplicate
		},
		{
			"2001:db8::1",
			"2001:db8::cafe",
		},
	}
	validAs = [][]string{
		{
			"192.0.2.1",
			"192.0.2.155",
		},
	}
	validA       = validAs[0]
	invalidAAAAs = [][]string{
		{}, // empty
		{
			"193.160.39.1",
		},
		{
			"2001:db8:::1",
			"2001:db8::a::1",
		},
		{
			"2001:db8::1",
			"2001:db8::1", // duplicate
		},
	}
	validAAAAs = [][]string{
		{
			"2001:db8::1",
			"2001:db8::cafe",
		},
		{
			"2001:db8::193.160.39.1",
		},
		{
			"3000::1",
			"::",
		},
	}
	validAAAA = validAAAAs[0]
)

func helperCompareRecords(a, b []*Record) bool {
	if len(a) != len(b) {
		return false
	}
	for _, recordA := range a {
		found := false
		for _, recordB := range b {
			if reflect.DeepEqual(recordA, recordB) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

/* --- Database ------------------------------------------------------------- */

func TestDBNew(t *testing.T) {
	db := New()
	assert.NotEqual(t, nil, db)
	assert.NotEqual(t, nil, db.root)
	assert.Equal(t, 0, len(db.root.children))
}

func TestDBNode(t *testing.T) {
	db := New()
	// invalid FQDNs
	for _, fqdn := range invalidFQDNs {
		nd, err := db.node(fqdn, false)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, (*node)(nil), nd)
	}
	// valid FQDNs without data
	for _, fqdn := range validFQDNs {
		nd, err := db.node(fqdn, false)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, (*node)(nil), nd)
	}
	// valid FQDNs to be created
	for _, fqdn := range validFQDNs {
		nd, err := db.node(fqdn, true)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, (*node)(nil), nd)
		if nd != nil {
			assert.Equal(t, fqdn, nd.fqdn)
			assert.NotEqual(t, nil, nd.parent)
		}
	}
	// parent pointers
	{
		db := New()
		ndFoo, err := db.node("foo.bar."+testFQDN, true)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, ndFoo)
		ndBar, err := db.node("bar."+testFQDN, false)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, ndBar)
		ndTest, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, ndTest)
		if ndFoo != nil && ndBar != nil && ndTest != nil {
			assert.Equal(t, ndBar, ndFoo.parent)
			assert.Equal(t, ndTest, ndBar.parent)
		}
	}
	// try to create a node below a delegation
	{
		db := New()
		err := db.SetNS(testFQDN, validTTL, validNS)
		assert.Equal(t, nil, err)
		err = db.SetAAAA("foo."+testFQDN, validTTL, validAAAA)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBRecords(t *testing.T) {
	// empty
	{
		db := New()
		records, err := db.Records(testFQDN, otherValidTTL)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, ([]*Record)(nil), records)
	}
	// regular records
	{
		db := New()
		_ = db.SetA(testFQDN, validTTL, validA)
		_ = db.SetAAAA(testFQDN, 0, validAAAA)
		for _, in := range validTXT.in {
			_ = db.AddTXT(testFQDN, validTTL, in)
		}
		_ = db.SetMX(testFQDN, 0, validMX)
		records, err := db.Records(testFQDN, otherValidTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, true, helperCompareRecords([]*Record{
				{
					FQDN:   testFQDN,
					RType:  "A",
					TTL:    validTTL,
					RDatas: validA,
				},
				{
					FQDN:   testFQDN,
					RType:  "AAAA",
					TTL:    otherValidTTL,
					RDatas: validAAAA,
				},
				{
					FQDN:   testFQDN,
					RType:  "TXT",
					TTL:    validTTL,
					RDatas: validTXT.out,
				},
				{
					FQDN:   testFQDN,
					RType:  "MX",
					TTL:    otherValidTTL,
					RDatas: validMX,
				},
			}, records))
		}
	}
	// NS
	{
		db := New()
		_ = db.SetNS(testFQDN, validTTL, validNS)
		records, err := db.Records(testFQDN, otherValidTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, true, helperCompareRecords([]*Record{
				{
					FQDN:   testFQDN,
					RType:  "NS",
					TTL:    validTTL,
					RDatas: validNS,
				},
			}, records))
		}
	}
	// CNAME
	{
		db := New()
		_ = db.SetCNAME(testFQDN, 0, validCNAME)
		records, err := db.Records(testFQDN, otherValidTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, true, helperCompareRecords([]*Record{
				{
					FQDN:   testFQDN,
					RType:  "CNAME",
					TTL:    validTTL * 2,
					RDatas: validCNAMEStr,
				},
			}, records))
		}
	}
}

func TestDBZone(t *testing.T) {
	// empty
	{
		db := New()
		records, err := db.Zone(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, ([]*Record)(nil), records)
	}
	// zone
	{
		db := New()

		_ = db.SetMX(testFQDN, validTTL, validMX)
		_ = db.SetNS("ns."+testFQDN, 0, validNS)
		_ = db.SetCNAME("cname."+testFQDN, 0, validCNAME)
		_ = db.SetA("foo."+testFQDN, validTTL, validA)
		_ = db.SetA("foo."+testFQDN, validTTL, validA)
		_ = db.SetAAAA("foo."+testFQDN, validTTL, validAAAA)
		for _, in := range validTXT.in {
			_ = db.AddTXT("foo."+testFQDN, validTTL, in)
		}
		records, err := db.Zone(testFQDN, otherValidTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, true, helperCompareRecords([]*Record{
				{
					FQDN:   testFQDN,
					RType:  "MX",
					TTL:    validTTL,
					RDatas: validMX,
				},
				{
					FQDN:   "ns." + testFQDN,
					RType:  "NS",
					TTL:    validTTL * 2,
					RDatas: validNS,
				},
				{
					FQDN:   "cname." + testFQDN,
					RType:  "CNAME",
					TTL:    validTTL * 2,
					RDatas: validCNAMEStr,
				},
				{
					FQDN:   "foo." + testFQDN,
					RType:  "A",
					TTL:    validTTL,
					RDatas: validA,
				},
				{
					FQDN:   "foo." + testFQDN,
					RType:  "AAAA",
					TTL:    validTTL,
					RDatas: validAAAA,
				},
				{
					FQDN:   "foo." + testFQDN,
					RType:  "TXT",
					TTL:    validTTL,
					RDatas: validTXT.out,
				},
			}, records))
		}
	}
}

/* --- Node ----------------------------------------------------------------- */

func TestNodeHasRType(t *testing.T) {
	{
		db := New()
		nd, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		err := db.SetNS(testFQDN, validTTL, validNS)
		assert.Equal(t, nil, err)
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, true, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		err := db.SetMX(testFQDN, validTTL, validMX)
		assert.Equal(t, nil, err)
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, true, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(testFQDN, validTTL, in)
			assert.Equal(t, nil, err)
		}
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, true, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		err := db.SetCNAME(testFQDN, validTTL, validCNAME)
		assert.Equal(t, nil, err)
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, true, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		err := db.SetA(testFQDN, validTTL, validA)
		assert.Equal(t, nil, err)
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, true, nd.hasA())
			assert.Equal(t, false, nd.hasAAAA())
		}
	}
	{
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, validAAAA)
		assert.Equal(t, nil, err)
		nd, err := db.node(testFQDN, false)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.NotEqual(t, nil, nd)
			assert.Equal(t, false, nd.hasNS())
			assert.Equal(t, false, nd.hasMX())
			assert.Equal(t, false, nd.hasTXT())
			assert.Equal(t, false, nd.hasCNAME())
			assert.Equal(t, false, nd.hasA())
			assert.Equal(t, true, nd.hasAAAA())
		}
	}
}

func TestNodeHasChildren(t *testing.T) {
	{
		db := New()
		nd, err := db.node("foo.test.", true)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, nd)
		if err == nil {
			assert.Equal(t, false, nd.hasChildren())
		}
		nd, err = db.node("test.", true)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, nd)
		if err == nil {
			assert.Equal(t, true, nd.hasChildren())
		}
	}
}

/* --- NS ------------------------------------------------------------------- */

func TestDBSetNSParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		err := db.SetNS(testFQDN, ttl, validNS)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, ttl := range validTTLs {
		db := New()
		err := db.SetNS(testFQDN, ttl, validNS)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetNSParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		err := db.SetNS(fqdn, validTTL, validNS)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		err := db.SetNS(fqdn, validTTL, validNS)
		assert.Equal(t, nil, err)
	}
	// try to overwrite
	{
		db := New()
		err := db.SetNS(testFQDN, validTTL, validNS)
		assert.Equal(t, nil, err)
		err = db.SetNS(testFQDN, validTTL, validNS)
		assert.NotEqual(t, nil, err)
	}
	// set NS of a node that has children
	{
		db := New()
		err := db.SetAAAA("foo."+testFQDN, validTTL, validAAAA)
		assert.Equal(t, nil, err)
		err = db.SetNS(testFQDN, validTTL, validNS)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBSetNSParameterRdatas(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidNSs {
		db := New()
		err := db.SetNS(testFQDN, validTTL, rdata)
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validNSs {
		db := New()
		err := db.SetNS(testFQDN, validTTL, rdata)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetNSLogicCheck1(t *testing.T) {
	{
		testForFailure := func(t *testing.T, db *RRDB) {
			err := db.SetNS(testFQDN, validTTL, validNS)
			assert.NotEqual(t, nil, err)
		}
		{
			db := New()
			err := db.SetMX(testFQDN, validTTL, validMX)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			for _, in := range validTXT.in {
				err := db.AddTXT(testFQDN, validTTL, in)
				assert.Equal(t, nil, err)
				testForFailure(t, db)
			}
		}
		{
			db := New()
			err := db.SetCNAME(testFQDN, validTTL, validCNAME)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetA(testFQDN, validTTL, validA)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetAAAA(testFQDN, validTTL, validAAAA)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
	}
}

func TestDBNS(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.NS(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.NS(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		err := db.SetNS(testFQDN, validTTL, validNS)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.NS(testFQDN, validTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "NS", record.RType)
				assert.Equal(t, validTTL, record.TTL)
				assert.Equal(t, validNS, record.RDatas)
			}
		}
	}
	// retrieval with default TTL
	{
		db := New()
		err := db.SetNS(testFQDN, 0, validNS)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.NS(testFQDN, otherValidTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "NS", record.RType)
				assert.Equal(t, otherValidTTL, record.TTL)
				assert.Equal(t, validNS, record.RDatas)
			}
		}
	}
}

/* --- MX ------------------------------------------------------------------- */

func TestDBSetMXParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		err := db.SetMX(testFQDN, ttl, validMX)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, ttl := range validTTLs {
		db := New()
		err := db.SetMX(testFQDN, ttl, validMX)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetMXParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		err := db.SetMX(fqdn, validTTL, validMX)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		err := db.SetMX(fqdn, validTTL, validMX)
		assert.Equal(t, nil, err)
	}
	// try to overwrite
	{
		db := New()
		err := db.SetMX(testFQDN, validTTL, validMX)
		assert.Equal(t, nil, err)
		err = db.SetMX(testFQDN, validTTL, validMX)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBSetMXParameterRdatas(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidMXs {
		db := New()
		err := db.SetMX(testFQDN, validTTL, rdata)
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validMXs {
		db := New()
		err := db.SetMX(testFQDN, validTTL, rdata)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetMXLogicCheck1(t *testing.T) {
	{
		testForFailure := func(t *testing.T, db *RRDB) {
			err := db.SetMX(testFQDN, validTTL, validMX)
			assert.NotEqual(t, nil, err)
		}
		{
			db := New()
			err := db.SetNS(testFQDN, validTTL, validNS)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetCNAME(testFQDN, validTTL, validCNAME)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
	}
}

func TestDBMX(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.MX(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.MX(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		err := db.SetMX(testFQDN, validTTL, validMX)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.MX(testFQDN, validTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "MX", record.RType)
				assert.Equal(t, validTTL, record.TTL)
				assert.Equal(t, validMX, record.RDatas)
			}
		}
	}
	// retrieval with default TTL
	{
		db := New()
		err := db.SetMX(testFQDN, 0, validMX)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.MX(testFQDN, otherValidTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "MX", record.RType)
				assert.Equal(t, otherValidTTL, record.TTL)
				assert.Equal(t, validMX, record.RDatas)
			}
		}
	}
}

/* --- TXT ------------------------------------------------------------------ */

func TestDBAddTXTParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(testFQDN, ttl, in)
			assert.NotEqual(t, nil, err)
		}
	}
	// valid
	for _, ttl := range validTTLs {
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(testFQDN, ttl, in)
			assert.Equal(t, nil, err)
		}
	}
	// add with changed TTL
	db := New()
	for idx, in := range validTXT.in {
		err := db.AddTXT(testFQDN, 300+idx, in)
		if idx == 0 {
			continue
		}
		assert.NotEqual(t, nil, err)
	}
}

func TestDBAddTXTParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(fqdn, validTTL, in)
			assert.NotEqual(t, nil, err)
		}
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(fqdn, validTTL, in)
			assert.Equal(t, nil, err)
		}
	}
}

func TestDBAddTXTParameterRdatas(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidTXTs {
		db := New()
		var err error
		for _, in := range rdata {
			err = db.AddTXT(testFQDN, validTTL, in)
			if err != nil {
				break
			}
		}
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validTXTs {
		db := New()
		for _, in := range rdata.in {
			err := db.AddTXT(testFQDN, validTTL, in)
			assert.Equal(t, nil, err)
		}
	}
}

func TestDBAddTXTLogicCheck1(t *testing.T) {
	{
		testForFailure := func(t *testing.T, db *RRDB) {
			err := db.AddTXT(testFQDN, validTTL, validTXT.in[0])
			assert.NotEqual(t, nil, err)
		}
		{
			db := New()
			err := db.SetNS(testFQDN, validTTL, validNS)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetCNAME(testFQDN, validTTL, validCNAME)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
	}
}

func TestDBTXT(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.TXT(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.TXT(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(testFQDN, validTTL, in)
			assert.Equal(t, nil, err)
		}
		record, err := db.TXT(testFQDN, validTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, testFQDN, record.FQDN)
			assert.Equal(t, "TXT", record.RType)
			assert.Equal(t, validTTL, record.TTL)
			assert.Equal(t, validTXT.out, record.RDatas)
		}
	}
	// retrieval
	{
		db := New()
		for _, in := range validTXT.in {
			err := db.AddTXT(testFQDN, 0, in)
			assert.Equal(t, nil, err)
		}
		record, err := db.TXT(testFQDN, otherValidTTL)
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, testFQDN, record.FQDN)
			assert.Equal(t, "TXT", record.RType)
			assert.Equal(t, otherValidTTL, record.TTL)
			assert.Equal(t, validTXT.out, record.RDatas)
		}
	}
}

/* --- CNAME ---------------------------------------------------------------- */

func TestDBSetCNAMEParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		err := db.SetCNAME(testFQDN, ttl, validCNAME)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, ttl := range validTTLs {
		db := New()
		err := db.SetCNAME(testFQDN, ttl, validCNAME)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetCNAMEParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		err := db.SetCNAME(fqdn, validTTL, validCNAME)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		err := db.SetCNAME(fqdn, validTTL, validCNAME)
		assert.Equal(t, nil, err)
	}
	// try to overwrite
	{
		db := New()
		err := db.SetCNAME(testFQDN, validTTL, validCNAME)
		assert.Equal(t, nil, err)
		err = db.SetCNAME(testFQDN, validTTL, validCNAME)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBSetCNAMEParameterRdata(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidCNAMEs {
		db := New()
		err := db.SetCNAME(testFQDN, validTTL, rdata)
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validCNAMEs {
		db := New()
		err := db.SetCNAME(testFQDN, validTTL, rdata)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetCNAMELogicCheck1(t *testing.T) {
	testForFailure := func(t *testing.T, db *RRDB) {
		err := db.SetCNAME(testFQDN, validTTL, validCNAME)
		assert.NotEqual(t, nil, err)
	}
	{
		db := New()
		err := db.SetNS(testFQDN, validTTL, validNS)
		assert.Equal(t, nil, err)
		testForFailure(t, db)
	}
	{
		db := New()
		err := db.SetMX(testFQDN, validTTL, validMX)
		assert.Equal(t, nil, err)
		testForFailure(t, db)
	}
	{
		db := New()
		err := db.AddTXT(testFQDN, validTTL, validTXT.in[0])
		assert.Equal(t, nil, err)
		testForFailure(t, db)
	}
	{
		db := New()
		err := db.SetA(testFQDN, validTTL, validA)
		assert.Equal(t, nil, err)
		testForFailure(t, db)
	}
	{
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, validAAAA)
		assert.Equal(t, nil, err)
		testForFailure(t, db)
	}
}

func TestDBCNAME(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.CNAME(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.CNAME(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		err := db.SetCNAME(testFQDN, validTTL, validCNAME)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.CNAME(testFQDN, validTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "CNAME", record.RType)
				assert.Equal(t, validTTL, record.TTL)
				assert.Equal(t, validCNAMEStr, record.RDatas)
			}
		}
	}
	// retrieval with default TTL
	{
		db := New()
		err := db.SetCNAME(testFQDN, 0, validCNAME)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.CNAME(testFQDN, otherValidTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "CNAME", record.RType)
				assert.Equal(t, otherValidTTL, record.TTL)
				assert.Equal(t, validCNAMEStr, record.RDatas)
			}
		}
	}
}

/* --- A -------------------------------------------------------------------- */

func TestDBSetAParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		err := db.SetA(testFQDN, ttl, validA)
		assert.NotEqual(t, nil, err)
	}
	for _, ttl := range validTTLs {
		// valid
		db := New()
		err := db.SetA(testFQDN, ttl, validA)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetAParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		err := db.SetA(fqdn, validTTL, validA)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		err := db.SetA(fqdn, validTTL, validA)
		assert.Equal(t, nil, err)
	}
	// try to overwrite
	{
		db := New()
		err := db.SetA(testFQDN, validTTL, validA)
		assert.Equal(t, nil, err)
		err = db.SetA(testFQDN, validTTL, validA)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBSetAParameterRdatas(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidAs {
		db := New()
		err := db.SetA(testFQDN, validTTL, rdata)
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validAs {
		db := New()
		err := db.SetA(testFQDN, validTTL, rdata)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetALogicCheck1(t *testing.T) {
	{
		testForFailure := func(t *testing.T, db *RRDB) {
			err := db.SetA(testFQDN, validTTL, validA)
			assert.NotEqual(t, nil, err)
		}
		{
			db := New()
			err := db.SetNS(testFQDN, validTTL, validNS)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetCNAME(testFQDN, validTTL, validCNAME)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
	}
}

func TestDBA(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.A(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.A(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		err := db.SetA(testFQDN, validTTL, validA)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.A(testFQDN, validTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "A", record.RType)
				assert.Equal(t, validTTL, record.TTL)
				assert.Equal(t, validA, record.RDatas)
			}
		}
	}
	// retrieval with defalt TTL
	{
		db := New()
		err := db.SetA(testFQDN, 0, validA)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.A(testFQDN, otherValidTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "A", record.RType)
				assert.Equal(t, otherValidTTL, record.TTL)
				assert.Equal(t, validA, record.RDatas)
			}
		}
	}
}

/* --- AAAA ----------------------------------------------------------------- */

func TestDBSetAAAAParameterTTL(t *testing.T) {
	// invalid
	for _, ttl := range invalidTTLs {
		db := New()
		err := db.SetAAAA(testFQDN, ttl, validAAAA)
		assert.NotEqual(t, nil, err)
	}
	for _, ttl := range validTTLs {
		// valid
		db := New()
		err := db.SetAAAA(testFQDN, ttl, validAAAA)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetAAAAParameterFQDN(t *testing.T) {
	// invalid
	for _, fqdn := range invalidFQDNs {
		db := New()
		err := db.SetAAAA(fqdn, validTTL, validAAAA)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		db := New()
		err := db.SetAAAA(fqdn, validTTL, validAAAA)
		assert.Equal(t, nil, err)
	}
	// try to overwrite
	{
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, validAAAA)
		assert.Equal(t, nil, err)
		err = db.SetAAAA(testFQDN, validTTL, validAAAA)
		assert.NotEqual(t, nil, err)
	}
}

func TestDBSetAAAAParameterRdatas(t *testing.T) {
	// invalid rdata
	for _, rdata := range invalidAAAAs {
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, rdata)
		assert.NotEqual(t, nil, err)
	}
	// valid rdata
	for _, rdata := range validAAAAs {
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, rdata)
		assert.Equal(t, nil, err)
	}
}

func TestDBSetAAAALogicCheck1(t *testing.T) {
	{
		testForFailure := func(t *testing.T, db *RRDB) {
			err := db.SetAAAA(testFQDN, validTTL, validAAAA)
			assert.NotEqual(t, nil, err)
		}
		{
			db := New()
			err := db.SetNS(testFQDN, validTTL, validNS)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
		{
			db := New()
			err := db.SetCNAME(testFQDN, validTTL, validCNAME)
			assert.Equal(t, nil, err)
			testForFailure(t, db)
		}
	}
}

func TestDBAAAA(t *testing.T) {
	// nonexistent
	{
		db := New()
		_, err := db.AAAA(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// empty
	{
		db := New()
		_, err := db.node(testFQDN, true)
		assert.Equal(t, nil, err)
		_, err = db.AAAA(testFQDN, validTTL)
		assert.NotEqual(t, nil, err)
	}
	// retrieval
	{
		db := New()
		err := db.SetAAAA(testFQDN, validTTL, validAAAA)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.AAAA(testFQDN, validTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "AAAA", record.RType)
				assert.Equal(t, validTTL, record.TTL)
				assert.Equal(t, validAAAA, record.RDatas)
			}
		}
	}
	// retrieval with default TTL
	{
		db := New()
		err := db.SetAAAA(testFQDN, 0, validAAAA)
		assert.Equal(t, nil, err)
		if err == nil {
			record, err2 := db.AAAA(testFQDN, otherValidTTL)
			assert.Equal(t, nil, err2)
			if err2 == nil {
				assert.Equal(t, testFQDN, record.FQDN)
				assert.Equal(t, "AAAA", record.RType)
				assert.Equal(t, otherValidTTL, record.TTL)
				assert.Equal(t, validAAAA, record.RDatas)
			}
		}
	}
}
