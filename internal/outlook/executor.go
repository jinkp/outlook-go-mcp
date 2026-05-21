//go:build windows

package outlook

import (
	"context"
	"runtime"
	"sync"
)

type Job struct {
	Fn     func() error
	Result chan error
}

type COMExecutor struct {
	jobs     chan Job
	session  OutlookSession
	done     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

func NewCOMExecutor(session OutlookSession) *COMExecutor {
	return &COMExecutor{
		jobs:    make(chan Job),
		session: session,
		done:    make(chan struct{}),
	}
}

// Start launches the COM worker goroutine WITHOUT connecting to Outlook.
// The connection is established lazily on the first Submit() call.
// This allows the MCP server to start successfully even when Outlook is not running.
func (e *COMExecutor) Start() error {
	e.wg.Add(1)
	go func() {
		runtime.LockOSThread()
		defer e.wg.Done()

		for {
			select {
			case <-e.done:
				_ = e.session.Close()
				return
			case job := <-e.jobs:
				// Connect lazily on first job if not already connected.
				if !e.session.IsConnected() {
					if err := e.session.Connect(); err != nil {
						job.Result <- err
						continue
					}
				}
				job.Result <- job.Fn()
			}
		}
	}()

	return nil // always succeeds — connection is deferred
}

func (e *COMExecutor) Stop() {
	e.stopOnce.Do(func() {
		close(e.done)
		e.wg.Wait()
	})
}

func (e *COMExecutor) Submit(ctx context.Context, fn func() error) error {
	job := Job{Fn: fn, Result: make(chan error, 1)}

	select {
	case <-e.done:
		return ErrNotConnected
	case <-ctx.Done():
		return ctx.Err()
	case e.jobs <- job:
	}

	select {
	case <-e.done:
		return ErrNotConnected
	case <-ctx.Done():
		return ctx.Err()
	case err := <-job.Result:
		return err
	}
}
