// Package main provides the mzdump tool which outputs a list of managed zones
// from Cloud DNS via customizable template.
package main

import (
	"flag"
	"html/template"
	"log"
	"os"

	"github.com/egymgmbh/dns-tools/gcp"
)

const defaultTemplate = `{{ range .ManagedZones }}
Managed Zone: {{ .DnsName }}
* Name: {{ .Name }}
* NameServers:
{{- range .NameServers }}
  * {{ . -}}
{{ end }}
{{ end }}`

func main() {
	gcpSAFile := flag.String("gcp-sa-file", "secret/gcp-sa.json",
		"Google Cloud Platform Service Account file in JSON format.")
	templateFile := flag.String("template-file", "",
		"Output template. See template.tmpl.example for details.")
	flag.Parse()

	service, projectID, err := gcp.GetDNSService(*gcpSAFile, true)
	if err != nil {
		log.Fatalf("DNS API service: %v", err)
	}

	tmpl, err := template.New("tmpl").Parse(defaultTemplate)
	if err != nil {
		log.Fatalf("default template: %v", err)
	}
	if *templateFile != "" {
		tmpl, err = template.ParseFiles(*templateFile)
		if err != nil {
			log.Fatalf("load template: %v", err)
		}
	}

	// fetch current managed zones
	managedZones, err := service.ManagedZones.List(projectID).Do()
	if err != nil {
		log.Fatalf("list managed zones: %v", err)
	}
	err = tmpl.Execute(os.Stdout, managedZones)
	if err != nil {
		log.Fatalf("execute template: %v", err)
	}
}
