package gcp

import (
	"path"
	"testing"

	"github.com/egymgmbh/dns-tools/rrdb"
	"github.com/stretchr/testify/assert"

	clouddns "google.golang.org/api/dns/v1"
)

func TestGetDNSService(t *testing.T) {
	{
		_, projectID, err := GetDNSService(path.Join("testdata", "okish-sa.json"), false)
		assert.Equal(t, nil, err)
		assert.Equal(t, "staging-co", projectID)
	}
	{
		_, projectID, err := GetDNSService(path.Join("testdata", "bad-sa.json"), false)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, "", projectID)
	}
	{
		_, projectID, err := GetDNSService(path.Join("testdata", "broken-sa.json"), false)
		assert.NotEqual(t, nil, err)
		assert.Equal(t, "", projectID)
	}
}

func TestRemoveNilPointersFromRRS(t *testing.T) {
	{
		in := []*clouddns.ResourceRecordSet{
			{
				Kind:    "dns#resourceRecordSet",
				Name:    "foo.test.",
				Type:    "TXT",
				Ttl:     300,
				Rrdatas: []string{"John", "Galt", "Line"},
			},
			{
				Kind:    "dns#resourceRecordSet",
				Name:    "foo.test.",
				Type:    "TXT",
				Ttl:     300,
				Rrdatas: []string{"John", "Galt", "Line"},
			},
		}
		out := in
		assert.Equal(t, out, removeNilPointersFromRRS(in))
	}
	{
		in := []*clouddns.ResourceRecordSet{
			{
				Kind:    "dns#resourceRecordSet",
				Name:    "foo.test.",
				Type:    "TXT",
				Ttl:     300,
				Rrdatas: []string{"John", "Galt", "Line"},
			},
			nil,
		}
		out := []*clouddns.ResourceRecordSet{
			in[0],
		}
		assert.Equal(t, out, removeNilPointersFromRRS(in))
	}
	{
		in := []*clouddns.ResourceRecordSet{
			nil,
			nil,
		}
		out := []*clouddns.ResourceRecordSet{}
		assert.Equal(t, out, removeNilPointersFromRRS(in))
	}
}

func TestRRDBRecordsToCloudDNSRecords(t *testing.T) {
	{
		in := []*rrdb.Record{
			{
				FQDN:   "foo.test.",
				RType:  "AAAA",
				TTL:    300,
				RDatas: []string{"2001:db8::1", "2001:db8:10::99"},
			},
		}
		out := []*clouddns.ResourceRecordSet{
			{
				Kind:    "dns#resourceRecordSet",
				Name:    "foo.test.",
				Type:    "AAAA",
				Ttl:     300,
				Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
			},
		}
		assert.Equal(t, out, RRDBRecordsToCloudDNSRecords(in))
	}
}

func TestRecordID(t *testing.T) {
	{
		in := clouddns.ResourceRecordSet{
			Kind:    "dns#resourceRecordSet",
			Name:    "foo.test.",
			Type:    "TXT",
			Ttl:     300,
			Rrdatas: []string{},
		}
		id := recordID(&in)
		assert.Equal(t, "dns#resourceRecordSet|foo.test.|TXT|300", id)
	}
}

func TestRemoveDuplicatesFromChange(t *testing.T) {
	// empty
	{
		in := clouddns.Change{
			Deletions: []*clouddns.ResourceRecordSet{},
			Additions: []*clouddns.ResourceRecordSet{},
		}
		out := in
		RemoveDuplicatesFromChange(&in)
		assert.Equal(t, out, in)
	}
	// must not change
	{
		in := clouddns.Change{
			Deletions: []*clouddns.ResourceRecordSet{
				{
					Kind:    "dns#resourceRecordSet",
					Name:    "foo.test.",
					Type:    "AAAA",
					Ttl:     300,
					Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
				},
			},
			Additions: []*clouddns.ResourceRecordSet{},
		}
		out := in
		RemoveDuplicatesFromChange(&in)
		assert.Equal(t, out, in)
	}
	{
		in := clouddns.Change{
			Deletions: []*clouddns.ResourceRecordSet{},
			Additions: []*clouddns.ResourceRecordSet{
				{
					Kind:    "dns#resourceRecordSet",
					Name:    "foo.test.",
					Type:    "AAAA",
					Ttl:     300,
					Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
				},
			},
		}
		out := in
		RemoveDuplicatesFromChange(&in)
		assert.Equal(t, out, in)
	}
	// remove one deletion and one addition
	{
		in := clouddns.Change{
			Deletions: []*clouddns.ResourceRecordSet{
				{
					Kind:    "dns#resourceRecordSet",
					Name:    "foo.test.",
					Type:    "AAAA",
					Ttl:     300,
					Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
				},
				{
					Kind:    "dns#resourceRecordSet",
					Name:    "foo2.test.",
					Type:    "AAAA",
					Ttl:     300,
					Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
				},
			},
			Additions: []*clouddns.ResourceRecordSet{
				{
					Kind:    "dns#resourceRecordSet",
					Name:    "foo.test.",
					Type:    "AAAA",
					Ttl:     300,
					Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
				},
			},
		}
		out := clouddns.Change{
			Deletions: []*clouddns.ResourceRecordSet{
				in.Deletions[1],
			},
			Additions: []*clouddns.ResourceRecordSet{},
		}
		RemoveDuplicatesFromChange(&in)
		assert.Equal(t, out, in)
	}
}

func TestFormatRRSets(t *testing.T) {
	{
		out := FormatRRSets([]*clouddns.ResourceRecordSet{
			{
				Kind:    "dns#resourceRecordSet",
				Name:    "foo.test.",
				Type:    "AAAA",
				Ttl:     300,
				Rrdatas: []string{"2001:db8::1", "2001:db8:10::99"},
			},
		})
		assert.Equal(t,
			[]string{"*foo.test. AAAA 300", " *2001:db8::1", " *2001:db8:10::99"},
			out)
	}
}

func TestFilterRRSets(t *testing.T) {
	// wrong kind
	{
		rrsets := []*clouddns.ResourceRecordSet{
			{
				Kind: "dns#foobar",
			},
		}
		filtered := FilterRRSets(rrsets, "foo.test.")
		assert.Equal(t, []*clouddns.ResourceRecordSet{}, filtered)
	}
	// don't touch the zone's NS record
	{
		rrsets := []*clouddns.ResourceRecordSet{
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "NS",
			},
		}
		filtered := FilterRRSets(rrsets, "foo.test.")
		assert.Equal(t, []*clouddns.ResourceRecordSet{}, filtered)
	}
	// good records
	{
		rrsets := []*clouddns.ResourceRecordSet{
			{
				Kind: "dns#resourceRecordSet",
				Name: "bar.foo.test.",
				Type: "NS",
			},
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "MX",
			},
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "TXT",
			},
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "CNAME",
			},
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "A",
			},
			{
				Kind: "dns#resourceRecordSet",
				Name: "foo.test.",
				Type: "AAAA",
			},
		}
		filtered := FilterRRSets(rrsets, "foo.test.")
		assert.Equal(t, rrsets, filtered)
	}
}
