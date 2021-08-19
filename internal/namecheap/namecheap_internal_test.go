package namecheap

import (
	"encoding/xml"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// This mostly tests the xml unmarshaling.

var (
	namecheapXMLNS = xml.Name{Space: "https://api.namecheap.com/xml.response", Local: "ApiResponse"}
)

const (
	setHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="https://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.setHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.setHosts">
    <DomainDNSSetHostsResult Domain="domain.com" IsSuccess="true" />
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	getHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="https://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.getHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.getHosts">
    <DomainDNSGetHostsResult Domain="domain.com" IsUsingOurDNS="true">
      <Host HostId="12" Name="@" Type="A" Address="1.2.3.4" MXPref="10" TTL="1800" />
      <Host HostId="14" Name="www" Type="A" Address="122.23.3.7" MXPref="10" TTL="1800" />
    </DomainDNSGetHostsResult>
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	emptyHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="https://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.getHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.getHosts">
    <DomainDNSGetHostsResult Domain="domain.com" IsUsingOurDNS="true" />
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	errorResponse = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="https://api.namecheap.com/xml.response">
  <Errors>
    <Error Number="1010102">Parameter APIKey is missing</Error>
  </Errors>
  <Warnings />
  <RequestedCommand />
  <Server>TEST111</Server>
  <GMTTimeDifference>--1:00</GMTTimeDifference>
  <ExecutionTime>0</ExecutionTime>
</ApiResponse>`
)

func TestUnmarshalAPIResponse(t *testing.T) {
	cases := map[string]struct {
		xmlResp  string
		expected apiResponse
	}{
		"getHosts with hosts": {
			xmlResp: getHostsResponse,
			expected: apiResponse{
				Status:           "OK",
				XMLName:          namecheapXMLNS,
				RequestedCommand: "namecheap.domains.dns.getHosts",
				Errors:           []apiError{},
				Server:           "SERVER-NAME",
				CommandResponse: commandResponse{
					Type: "namecheap.domains.dns.getHosts",
					DomainDNSGetHostsResult: &domainDNSGetHostsResult{
						Domain:        "domain.com",
						IsUsingOurDNS: true,
						Hosts: []getHostsResponseRecord{
							{
								HostID:  "12",
								Name:    "@",
								Type:    "A",
								Address: "1.2.3.4",
								MXPref:  "10",
								TTL:     1800,
							},
							{
								HostID:  "14",
								Name:    "www",
								Type:    "A",
								Address: "122.23.3.7",
								MXPref:  "10",
								TTL:     1800,
							},
						},
					},
				},
			},
		},
		"getHosts without hosts": {
			xmlResp: emptyHostsResponse,
			expected: apiResponse{
				Status:           "OK",
				XMLName:          namecheapXMLNS,
				RequestedCommand: "namecheap.domains.dns.getHosts",
				Errors:           []apiError{},
				Server:           "SERVER-NAME",
				CommandResponse: commandResponse{
					Type: "namecheap.domains.dns.getHosts",
					DomainDNSGetHostsResult: &domainDNSGetHostsResult{
						Domain:        "domain.com",
						IsUsingOurDNS: true,
					},
				},
			},
		},
		"setHosts": {
			xmlResp: setHostsResponse,
			expected: apiResponse{
				Status:           "OK",
				XMLName:          namecheapXMLNS,
				RequestedCommand: "namecheap.domains.dns.setHosts",
				Errors:           []apiError{},
				Server:           "SERVER-NAME",
				CommandResponse: commandResponse{
					Type: "namecheap.domains.dns.setHosts",
					DomainDNSSetHostsResult: &domainDNSSetHostsResult{
						Domain:    "domain.com",
						IsSuccess: true,
					},
				},
			},
		},
		"Response with error": {
			xmlResp: errorResponse,
			expected: apiResponse{
				Status:  "ERROR",
				XMLName: namecheapXMLNS,
				Errors: []apiError{
					{
						Err:    "Parameter APIKey is missing",
						Number: "1010102",
					},
				},
				Server: "TEST111",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var r apiResponse
			if err := xml.Unmarshal([]byte(tc.xmlResp), &r); err != nil {
				t.Fatalf("Unexpected error while unmarshaling. Err: %s", err)
			}

			if diff := cmp.Diff(tc.expected, r); diff != "" {
				t.Fatalf("Unmarshaling has unexpected diff: %s", diff)
			}
		})
	}
}
