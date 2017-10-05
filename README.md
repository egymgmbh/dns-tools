# dns-tools

[![Build Status](https://travis-ci.org/egymgmbh/dns-tools.svg?branch=master)](https://travis-ci.org/egymgmbh/dns-tools)
[![Go Report Card](https://goreportcard.com/badge/github.com/egymgmbh/dns-tools)](https://goreportcard.com/report/github.com/egymgmbh/dns-tools)

The dns-tools parse resource record and zone information from YAML-formatted
plain text files and perform various checks and actions on them. They use the
Google Cloud DNS API to do monitoring, checks, and changes. The dns-tools are
written in Golang and like to be part of an automation pipeline.


## Tools

* `dbcheck` Loads zone data from a local directory into a in-memory database and
  verifies that all configured managed zones are retrievable from that database.
  We use this tool to test our zone data changes in a continuous deployment
  pipeline.
* `mzcreate` Connects to a Google Cloud Platform project and creates all managed
  zones that are in the local configuration file but missing on CloudDNS.
  We use this tool to easily add new zones by just adding them in the YAML files
  and letting our continuous integration deployment take care of the rest.
* `mzdump` Fetches all CloudDNS managed zones from a Google Cloud Platform
  project and prints the associated meta data, such as nameservers, using a
  template.
  We used this tool to create [JIRA](https://www.atlassian.com/software/jira)
  tickets for nameserver changes using JIRA's CSV import function.
* `mzmon` Fetches all CloudDNS managed zones from a Google Cloud Platform
  project and uses DNS lookups to check if delegations are correctly implemented
  at the corresponding Top Level Domain (TLD) nameservers. The results are then
  written to an InfluxDB time series database in a configurable interval.
  We use this tool to continuously monitor the delegations of our
  business-critical domains.
* `rrlookup` Loads zone data from a local directory into a in-memory database
  and looks up all records of all configured managed zones and compares the
  results with the expected values from the database.
  We use this tool to check for DNS errors and to verify our deployments went
  well.
* `rrpush` Loads zone data from a local directory into a in-memory database
  and deploys all records of all configured managed zones to CloudDNS.
  We use this tool to rapidly deploy changes to our zone data repository into
  production.


## Configuration

### dns-tools

The configuration file, default filename `config.yml`, consists of the dns-tools
configuration. It includes a path a directory that holds the zone data in form
of YAML files. The default TTL used for all records of all managed zones is also
configured here. The most important part is the list of managed zones that the
dns-tools are allowed to touch. It is possible to set a zone-specific TTL value
if needed. The dns-tools are most useful if you have hundreds of zones and
thousands of records.

    config:
      zonedatadirectory: zonedata
      defaults:
        ttl: 300
      managedzones:
      - fqdn: example.com.
        ttl: 3600
      - fqdn: example.org.


### InfluxDB

Some tools may use InfluxDB to submit metrics. These tools expect an InfluxDB
configuration file in JSON format.

    {
    	"server": "https://influx.example.com:8086/",
    	"database": "dns",
    	"username": "dns",
    	"password": "topsecret"
    }


## Zone Data

The dns-tools read from what should be the single source of truth for zone
information. The zone information consists of *zones* and *templates*, both
being a container for *names*. Names contain the actual data eventually
resulting in a DNS resource record. If a zone references a template (we say it
*pulls in* a template) all the names from that template are added to the zone
as if they were defined in that zone. A template may be used by multiple zones
and a zone may be pulling in multiple templates. Resource records derived from
templates do not overwrite each other but add up (if allowed) or fail the
built-in sanity checks. For example, it is common to have multiple text records
for a zone of which some originate from the zone definition while others have
been pulled in via a template.


    +-----------+       +-----------------+
    | +-----------+     | Service Account |
    | | +-----------+   | File (JSON)     |
    | | |           |   +-----------------+
    +-| | Zone Data |            |                 ,--.
      +-| (YAML)    |            v             _.-(    )-.
        |           |------> dns-tools <====> (_Cloud_DNS_)
        +-----------+            ^
                                 |
                        +-----------------+
                        | Configuration   |
                        | File (YAML)     |
                        +-----------------+

The dns-tools intentionally do not allow full flexibility over resource records
but abstract away some technical details. It is therefore not a tool suitable
for everyone. Human-friendly resource record definitions are preferred over
extensive manipulation options. We like changes in resource record definitions
to be reviewed by humans with ease, so we made some assumptions that you or your
organization may not share.


### Template Example

````
templates:
  - template: gmail
    description: >
      This template adds Google mailservers to a zone.
    names:
      - name: '@'
        mail:
          ttl: 604800 # 1 week = 604800 seconds
          mailservers:
            - mailserver: aspmx.l.google.com.
              priority: 10
            - mailserver: alt1.aspmx.l.google.com.
              priority: 20
            - mailserver: alt2.aspmx.l.google.com.
              priority: 20
            - mailserver: aspmx2.googlemail.com.
              priority: 30
            - mailserver: aspmx3.googlemail.com.
              priority: 30
            - mailserver: aspmx4.googlemail.com.
              priority: 30
            - mailserver: aspmx5.googlemail.com.
              priority: 30
      - name: google._domainkey
        texts:
          data:
            - >
              v=DKIM1;
              k=rsa;
              p=foobar123456
  - template: website
    description: Our consumer facing website.
    names:
      - name: '@'
        addresses:
          literals:
            - 192.0.2.1
            - 198.51.100.1
            - 2001:db8::1
            - 2001:db8::babe
      - name: www
        forwarding:
          target: '@'
````


### Zone Example

````
zones:
  - zone: egym.coffee.
    description: Test zone.
    ttl: 300
    templates:
      - gmail
      - website
    names:
      - name: '@'
        texts:
          data:
            - foobar-site-verification-123456
      - name: paloalto
        forwarding:
          ttl: 60
          target: flaky.cloud.example.com.
      - name: losangeles
        addresses:
          literals:
            - 2001:db8:100::99
            - 2001:db8:200::99
        texts:
          data:
            - Oompa Loompas
            - Chocolate
            - A third TXT record, just for fun!
      - name: subdomain
        delegation:
          ttl: 3600
          nameservers:
            - ns1.example.com.
            - ns2.example.com.
````

## About RRDB

The Resource Record Database (RRDB) is an in-memory, trie-based database that
holds the DNS tree starting from the root label. The trie can be walked label
by label starting from the root label. Resource record data is attached to
nodes. Before creating a new node, resource record data of parent nodes can be
checked to detect logical errors. For example, if a parent record holds a
delegation (that is, NS records), a potential child node can not be created as
it would be outside of the authority of that zone. Similarly, a delegation can
not be assigned to a node that has child nodes.
Certain resource record types run additional checks to prevent common logic
errors. Additional checks can be added easily in the corresponding code
sections.

## License

    Copyright 2017 eGym GmbH <support@egym.de>
    Copyright 2017 Dan Luedtke <dan.luedtke@egym.de>

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
