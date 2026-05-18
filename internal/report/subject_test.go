package report

import "testing"

func TestNormalizeSubjectStripesPrefixes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "RE prefix", input: "RE: Hello", want: "HELLO"},
		{name: "RES prefix", input: "RES: meeting notes", want: "MEETING NOTES"},
		{name: "FW prefix", input: "FW: Update", want: "UPDATE"},
		{name: "FWD prefix", input: "FWD: Report", want: "REPORT"},
		{name: "RV prefix", input: "RV: Review", want: "REVIEW"},
		{name: "AW prefix", input: "AW: Antwort", want: "ANTWORT"},
		{name: "WG prefix", input: "WG: Weitergeleitet", want: "WEITERGELEITET"},
		{name: "TR prefix", input: "TR: Transfer", want: "TRANSFER"},
		{name: "REENVIAR prefix", input: "REENVIAR: Correo", want: "CORREO"},
		{name: "REENV prefix", input: "REENV: Mensaje", want: "MENSAJE"},
		{name: "nested RE RE", input: "RES: RE: meeting", want: "MEETING"},
		{name: "nested FWD RV", input: "FWD: RV: Update", want: "UPDATE"},
		{name: "empty string", input: "", want: "(NO SUBJECT)"},
		{name: "no prefix", input: "No prefix", want: "NO PREFIX"},
		{name: "case insensitive lowercase re", input: "re: Hello", want: "HELLO"},
		{name: "case insensitive mixed case", input: "Re: Meeting", want: "MEETING"},
		{name: "only prefix whitespace", input: "RE:    ", want: "(NO SUBJECT)"},
		{name: "triple nested", input: "RE: RE: RE: deep", want: "DEEP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSubject(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeSubject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
