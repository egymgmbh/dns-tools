package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	invalidLookups := []struct {
		fqdn  string
		rtype string
	}{
		{
			fqdn:  "non-existent.dns-tools.egym.coffee.",
			rtype: "AAAA",
		},
		{
			fqdn:  "target.dns-tools.egym.coffee.",
			rtype: "FOOBAR",
		},
		{
			fqdn:  "-invalid.dns-tools.egym.coffee.",
			rtype: "AAAA",
		},
	}
	validLookups := []struct {
		fqdn  string
		rtype string
		out   []string
	}{
		{
			fqdn:  "delegations.dns-tools.egym.coffee.",
			rtype: "NS",
			out: []string{
				"ns-cloud-e1.googledomains.com.",
				"ns-cloud-e2.googledomains.com.",
				"ns-cloud-e3.googledomains.com.",
				"ns-cloud-e4.googledomains.com.",
			},
		},
		{
			fqdn:  "mailservers.dns-tools.egym.coffee.",
			rtype: "MX",
			out: []string{
				"10 mail1.example.com.",
				"666 mail2.example.com.",
				"500 mail3.example.com.",
			},
		},
		{
			fqdn:  "texts.dns-tools.egym.coffee.",
			rtype: "TXT",
			out: []string{
				"Life tasted so good, dude!",
				"All watched over by machines of love and grace...",
			},
		},
		{
			fqdn:  "forwarding.dns-tools.egym.coffee.",
			rtype: "CNAME",
			out:   []string{"target.dns-tools.egym.coffee."},
		},
		{
			fqdn:  "target.dns-tools.egym.coffee.",
			rtype: "A",
			out:   []string{"192.0.2.1"},
		},
		{
			fqdn:  "target.dns-tools.egym.coffee.",
			rtype: "AAAA",
			out:   []string{"2001:67c:26f4::1"},
		},
	}
	for _, test := range validLookups {
		rdatas, err := Lookup(test.fqdn, test.rtype)
		assert.Equal(t, nil, err)
		assert.Equal(t, len(test.out), len(rdatas))
		for _, rdata := range rdatas {
			assert.Contains(t, test.out, rdata)
		}
	}
	for _, test := range invalidLookups {
		_, err := Lookup(test.fqdn, test.rtype)
		assert.NotEqual(t, nil, err)
	}
}

func TestRDatasEqual(t *testing.T) {
	a1 := []string{
		"foo",
		"bar",
		"foo.bar.com.",
	}
	a2 := []string{
		"bar",
		"foo",
		"foo.bar.com.",
	}
	b := []string{
		"foo bar",
		"bar",
		"foo.bar.com",
	}
	c := []string{
		"foo bar",
		"bar",
	}

	d1 := []string{
		"foo bar",
		"bar",
		"bar",
	}
	d2 := []string{
		"foo bar",
		"foo bar",
		"bar",
	}

	assert.Equal(t, true, RDatasEqual([]string{}, []string{}))
	assert.Equal(t, true, RDatasEqual(a1, a1))
	assert.Equal(t, true, RDatasEqual(a2, a2))
	assert.Equal(t, true, RDatasEqual(a1, a2))
	assert.Equal(t, true, RDatasEqual(b, b))
	assert.Equal(t, true, RDatasEqual(c, c))
	assert.Equal(t, true, RDatasEqual(d1, d1))
	assert.Equal(t, true, RDatasEqual(d2, d2))

	assert.NotEqual(t, true, RDatasEqual(a1, b))
	assert.NotEqual(t, true, RDatasEqual(a2, b))
	assert.NotEqual(t, true, RDatasEqual(b, c))
	assert.NotEqual(t, true, RDatasEqual(d1, d2))
}

func TestMakeFQDN(t *testing.T) {
	{
		fqdn := MakeFQDN("foo", "test")
		assert.Equal(t, "foo.test", fqdn)
	}
	{
		fqdn := MakeFQDN("foo.", "test.")
		assert.Equal(t, "foo.", fqdn)
	}
	{
		fqdn := MakeFQDN("@", "test.")
		assert.Equal(t, "test.", fqdn)
	}
	{
		fqdn := MakeFQDN("foo", "test.")
		assert.Equal(t, "foo.test.", fqdn)
	}
}

func TestIsValidIPv4(t *testing.T) {
	invalidIPv4s := []string{
		"",
		"1.2.3",
		"192.0.2.256",
		"2001:db8::1",
		"2001:db8::cafe",
	}
	validIPv4s := []string{
		"192.0.2.1",
		"192.0.2.155",
	}
	for _, address := range invalidIPv4s {
		assert.NotEqual(t, nil, IsValidIPv4(address))
	}
	for _, address := range validIPv4s {
		assert.Equal(t, nil, IsValidIPv4(address))
	}
}

func TestIsValidIPv6(t *testing.T) {
	invalidIPv6s := []string{
		"",
		"193.160.39.1",
		"2001:db8:::1",
		"2001:db8::a::1",
		"2001:db8::a::g",
	}
	validIPv6s := []string{
		"2001:db8::1",
		"2001:db8::cafe",
		"2001:db8::193.160.39.1",
		"3000::1",
		"::",
	}
	for _, address := range invalidIPv6s {
		assert.NotEqual(t, nil, IsValidIPv6(address))
	}
	for _, address := range validIPv6s {
		assert.Equal(t, nil, IsValidIPv6(address))
	}
}

func TestIsValidTTL(t *testing.T) {
	invalidTTLs := []int{-300, -3, -2, -1, 2147483648, 2147483649, 2147483650}
	validTTLs := []int{0, 1, 2, 3, 10, 300, 2147483645, 2147483646, 2147483647}
	// invalid
	for _, ttl := range invalidTTLs {
		err := IsValidTTL(ttl)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, ttl := range validTTLs {
		err := IsValidTTL(ttl)
		assert.Equal(t, nil, err)
	}
}

func TestIsValidFQDN(t *testing.T) {
	invalidFQDNs := []string{
		"",
		".",
		"example.com",
		"-foo.example.com",
		"-foo.example.com.",
		"--foo.example.com.",
		"foo-.example.com.",
		"foo--.example.com.",
	}
	validFQDNs := []string{
		"com.",
		"example.com.",
		"foo.example.com.",
		"f-o-o.example.com.",
		"xn--foo-bar.example.com.",
		"_dmarc.example.com.",
		"_759c7c23f786497a69e4f2ea4eb6c329.example.com.",
		"fefa2c1f7d6ed2027ab0bf42326b3fc2.fd47d3d85924bd523093122a6add7633.example.com.",
	}
	// invalid
	for _, fqdn := range invalidFQDNs {
		err := IsValidFQDN(fqdn)
		assert.NotEqual(t, nil, err)
	}
	// valid
	for _, fqdn := range validFQDNs {
		err := IsValidFQDN(fqdn)
		assert.Equal(t, nil, err)
	}
}

func TestTextToQuotedStrings(t *testing.T) {
	quotedTexts := []struct {
		in  string
		out string
	}{
		{
			in:  "Free drinks at the Foo bar!",
			out: "\"Free\" \"drinks\" \"at\" \"the\" \"Foo\" \"bar!\"",
		},
		{
			in:  "So long and thanks for all the fish...",
			out: "\"So\" \"long\" \"and\" \"thanks\" \"for\" \"all\" \"the\" \"fish...\"",
		},
		{
			in:  "All watched over by machines of love and grace",
			out: "\"All\" \"watched\" \"over\" \"by\" \"machines\" \"of\" \"love\" \"and\" \"grace\"",
		},
		{
			in:  "",
			out: "",
		},
		{
			in:  " ",
			out: "",
		},
		{
			in:  "foo",
			out: "\"foo\"",
		},
	}
	for _, s := range quotedTexts {
		assert.Equal(t, s.out, TextToQuotedStrings(s.in))
	}
}
