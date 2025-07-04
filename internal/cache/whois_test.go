package cache

import (
	"testing"
)

func TestExtractWhoisSummary(t *testing.T) {
	tests := []struct {
		name      string
		whoisText string
		expected  string
	}{
		{
			name: "cloudflare whois",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      1.1.1.0 - 1.1.1.255
netname:      CLOUDFLARENET
descr:        Cloudflare, Inc.
country:      US
admin-c:      CLOUD14-ARIN
tech-c:       CLOUD14-ARIN`,
			expected: "CLOUDFLARE",
		},
		{
			name: "digitalocean whois",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      159.89.0.0 - 159.89.255.255
netname:      DIGITALOCEAN-159-89-0-0
descr:        DigitalOcean, LLC
country:      US`,
			expected: "DIGITALOCEAN",
		},
		{
			name: "amazon whois",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      52.0.0.0 - 52.15.255.255
netname:      AT-88-Z
descr:        Amazon Technologies Inc.
country:      US`,
			expected: "AMAZON",
		},
		{
			name: "country code extraction",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      203.0.113.0 - 203.0.113.255
netname:      TEST-NET-3
descr:        Documentation
country:      AU
admin-c:      IANA1-ARIN`,
			expected: "AU",
		},
		{
			name: "organization extraction",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      198.51.100.0 - 198.51.100.255
netname:      TEST-NET-2
descr:        Documentation
org:          Example Organization
country:      US`,
			expected: "Example",
		},
		{
			name: "fallback to first word",
			whoisText: `% IANA WHOIS server
% for more information on IANA, visit http://www.iana.org
% This query returned 1 object

refer:        whois.arin.net

inetnum:      192.0.2.0 - 192.0.2.255
netname:      TEST-NET-1
descr:        Documentation
country:      US`,
			expected: "US",
		},
		{
			name:      "empty whois text",
			whoisText: "",
			expected:  "whois",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWhoisSummary(tt.whoisText)
			if result != tt.expected {
				t.Errorf("extractWhoisSummary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Note: Tests for EnrichIPs are not included here because they would require
// real whois lookups which can hang or take a very long time. In a real
// testing environment, you would mock the whois.Whois function or use
// integration tests with a controlled whois server.
