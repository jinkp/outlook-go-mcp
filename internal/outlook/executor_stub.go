//go:build !windows

package outlook

import "context"

type Job struct {
	Fn     func() error
	Result chan error
}

type COMExecutor struct{}

func NewCOMExecutor(session OutlookSession) *COMExecutor {
	return &COMExecutor{}
}

func (e *COMExecutor) Start() error {
	return ErrNotConnected
}

func (e *COMExecutor) Stop() {}

func (e *COMExecutor) Submit(ctx context.Context, fn func() error) error {
	return ErrNotConnected
}
