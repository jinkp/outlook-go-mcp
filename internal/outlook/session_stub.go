//go:build !windows

package outlook

type outlookSession struct{}

func NewOutlookSession() OutlookSession {
	return &outlookSession{}
}

func (s *outlookSession) Connect() error {
	return ErrNotConnected
}

func (s *outlookSession) Close() error {
	return ErrNotConnected
}

func (s *outlookSession) IsConnected() bool {
	return false
}
