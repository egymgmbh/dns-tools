package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
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
)

func TestCheckTTL(t *testing.T) {
	// https://tools.ietf.org/html/rfc2181#section-8
	// invalid
	{
		err := checkTTL(-1)
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "invalid TTL: -1", err.Error())
		}
	}
	{
		err := checkTTL(0)
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "invalid TTL: 0", err.Error())
		}
	}
	{
		err := checkTTL(2147483648)
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "invalid TTL: 2147483648", err.Error())
		}
	}
	// valid
	{
		err := checkTTL(1)
		assert.Equal(t, nil, err)
	}
	{
		err := checkTTL(300)
		assert.Equal(t, nil, err)
	}
	{
		err := checkTTL(2147483647)
		assert.Equal(t, nil, err)
	}
}

func TestCheckFQDN(t *testing.T) {
	// invalid
	{
		for _, fqdn := range invalidFQDNs {
			err := checkFQDN(fqdn)
			assert.NotEqual(t, nil, err)
			if err != nil {
				assert.Equal(t, fmt.Sprintf("invalid FQDN: %v", fqdn), err.Error())
			}
		}
	}
	// valid
	{
		{
			for _, fqdn := range validFQDNs {
				err := checkFQDN(fqdn)
				assert.Equal(t, nil, err)
			}
		}
	}
}

func TestNew(t *testing.T) {
	// invalid filename
	{
		_, err := New("testdata/nonexistent.yml")
		assert.NotEqual(t, nil, err)
	}
	// broken format
	{
		_, err := New("testdata/broken-format.yml")
		assert.NotEqual(t, nil, err)
	}
	// invalid configuration
	{
		_, err := New("testdata/invalid-defaults-ttl.yml")
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "defaults: invalid TTL: 2147483650", err.Error())
		}
	}
	{
		_, err := New("testdata/invalid-fqdn.yml")
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "managed zone egym.de: invalid FQDN: egym.de", err.Error())
		}
	}
	{
		_, err := New("testdata/invalid-mz-ttl.yml")
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "managed zone egym.de.: invalid TTL: -1", err.Error())
		}
	}
	{
		_, err := New("testdata/duplicate-mz.yml")
		assert.NotEqual(t, nil, err)
		if err != nil {
			assert.Equal(t, "managed zone egym.de.: duplicate entry", err.Error())
		}
	}
	// valid configuration
	{
		config, err := New("testdata/complete.yml")
		assert.Equal(t, nil, err)
		if err == nil {
			assert.Equal(t, "zonedata/", config.ZoneDataDirectory)
			assert.Equal(t, 300, config.Defaults.TTL)
			assert.Equal(t, "egym.de.", config.ManagedZones[0].FQDN)
			assert.Equal(t, 1337, config.ManagedZones[0].TTL)
			assert.Equal(t, "egym.com.", config.ManagedZones[1].FQDN)
			assert.Equal(t, 300, config.ManagedZones[1].TTL)
		}
	}
}
