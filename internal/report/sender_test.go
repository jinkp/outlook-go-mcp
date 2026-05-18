package report

import "testing"

func TestMatchesVIP(t *testing.T) {
	tests := []struct {
		name string
		from string
		list []string
		want bool
	}{
		{name: "exact match", from: "boss@acme.com", list: []string{"boss@acme.com"}, want: true},
		{name: "domain match", from: "anyone@partner.com", list: []string{"@partner.com"}, want: true},
		{name: "different domain", from: "someone@other.com", list: []string{"@partner.com"}, want: false},
		{name: "empty list", from: "boss@acme.com", list: []string{}, want: false},
		{name: "nil list", from: "boss@acme.com", list: nil, want: false},
		{name: "case insensitive exact", from: "BOSS@ACME.COM", list: []string{"boss@acme.com"}, want: true},
		{name: "case insensitive domain", from: "ANYONE@PARTNER.COM", list: []string{"@partner.com"}, want: true},
		{name: "no match in list", from: "stranger@unknown.com", list: []string{"boss@acme.com", "@partner.com"}, want: false},
		{name: "match second entry", from: "user@partner.com", list: []string{"boss@acme.com", "@partner.com"}, want: true},
		{name: "exact beats domain check", from: "boss@acme.com", list: []string{"boss@acme.com", "@other.com"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesVIP(tt.from, tt.list)
			if got != tt.want {
				t.Fatalf("MatchesVIP(%q, %v) = %v, want %v", tt.from, tt.list, got, tt.want)
			}
		})
	}
}
