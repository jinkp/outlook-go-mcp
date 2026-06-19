//go:build windows

package outlook

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	// maxCOMRetries is the number of retry attempts when a COM call fails
	// with ErrCOMFailure. Between each retry the session is closed and
	// reconnected with exponential backoff.
	maxCOMRetries = 3

	// comRetryBaseDelay is the initial delay between retries. Each
	// subsequent retry doubles the delay (1s, 2s, 4s).
	comRetryBaseDelay = 1 * time.Second
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

		// PanicOnFault converts hardware faults (access violations, segfaults)
		// into recoverable Go panics. Without this, a stale COM pointer
		// dereference (0xc0000005) kills the entire process. With it, safeCall
		// can recover the panic and return ErrCOMFailure for retry.
		debug.SetPanicOnFault(true)

		for {
			select {
			case <-e.done:
				_ = e.session.Close()
				return
			case job := <-e.jobs:
				job.Result <- e.executeWithRetry(job.Fn)
			}
		}
	}()

	return nil // always succeeds — connection is deferred
}

// executeWithRetry runs fn after ensuring the session is connected. If fn
// returns an error wrapping ErrCOMFailure (including panics recovered by
// safeCall), the session is torn down, reconnected after a backoff delay,
// and fn is retried up to maxCOMRetries times.
//
// Non-COM errors (validation, not-found, policy) are returned immediately
// without retry — they would fail again with the same input.
func (e *COMExecutor) executeWithRetry(fn func() error) error {
	for attempt := range maxCOMRetries {
		// Ensure connected (lazy connect on first call, reconnect on retry).
		if !e.session.IsConnected() {
			if err := e.session.Connect(); err != nil {
				if attempt == maxCOMRetries-1 {
					return err
				}
				time.Sleep(comRetryBaseDelay << uint(attempt))
				continue
			}
		}

		err := safeCall(fn)
		if err == nil {
			return nil
		}

		// Only retry on COM failures (stale session, disconnected server, etc).
		if !errors.Is(err, ErrCOMFailure) {
			return err
		}

		// Last attempt — return the error as-is.
		if attempt == maxCOMRetries-1 {
			return err
		}

		// Tear down the stale session and retry after backoff.
		_ = e.session.Close()
		time.Sleep(comRetryBaseDelay << uint(attempt))
	}

	return ErrNotConnected // unreachable, but satisfies the compiler
}

// safeCall executes fn and recovers from COM-induced panics (e.g. access
// violations when the Exchange server disconnects mid-call). The recovered
// panic is converted to a regular error so the worker goroutine survives.
func safeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: COM panic recovered: %v", ErrCOMFailure, r)
		}
	}()
	return fn()
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
