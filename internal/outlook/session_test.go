package outlook

import "testing"

var _ OutlookSession = (*outlookSession)(nil)

func TestOutlookSessionSatisfiesInterface(t *testing.T) {
	session := NewOutlookSession()
	if session == nil {
		t.Fatal("NewOutlookSession() returned nil")
	}
}
