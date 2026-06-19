//go:build windows

package outlook

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type outlookSession struct {
	ole       *ole.IDispatch
	mapi      *ole.IDispatch
	mu        sync.Mutex
	connected bool
}

func NewOutlookSession() OutlookSession {
	return &outlookSession{}
}

func (s *outlookSession) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	runtime.LockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return fmt.Errorf("%w: initialize COM apartment: %v", ErrCOMFailure, err)
	}

	appUnknown, err := oleutil.GetActiveObject("Outlook.Application")
	if err != nil {
		appUnknown, err = oleutil.CreateObject("Outlook.Application")
		if err != nil {
			ole.CoUninitialize()
			return fmt.Errorf("%w: connect to Outlook application: %v", ErrNotConnected, err)
		}
	}
	defer appUnknown.Release()

	appDispatch, err := appUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		ole.CoUninitialize()
		return fmt.Errorf("%w: query Outlook dispatch: %v", ErrCOMFailure, err)
	}

	mapiVariant, err := oleutil.CallMethod(appDispatch, "GetNamespace", "MAPI")
	if err != nil {
		appDispatch.Release()
		ole.CoUninitialize()
		return fmt.Errorf("%w: get MAPI namespace: %v", ErrCOMFailure, err)
	}
	defer mapiVariant.Clear()

	mapiDispatch := mapiVariant.ToIDispatch()
	if mapiDispatch == nil {
		appDispatch.Release()
		ole.CoUninitialize()
		return fmt.Errorf("%w: get MAPI namespace returned nil dispatch", ErrCOMFailure)
	}

	// Validate the MAPI namespace is functional by accessing the Inbox folder
	// name. Some environments return a valid-looking dispatch that is actually
	// disconnected from Exchange. This catches it early, before we mark the
	// session as connected.
	inboxVariant, err := oleutil.CallMethod(mapiDispatch, "GetDefaultFolder", 6) // olFolderInbox
	if err != nil {
		mapiDispatch.Release()
		appDispatch.Release()
		ole.CoUninitialize()
		return fmt.Errorf("%w: MAPI validation failed (GetDefaultFolder): %v", ErrCOMFailure, err)
	}
	inboxDispatch := inboxVariant.ToIDispatch()
	if inboxDispatch == nil {
		mapiDispatch.Release()
		appDispatch.Release()
		ole.CoUninitialize()
		return fmt.Errorf("%w: MAPI validation failed: GetDefaultFolder returned nil", ErrCOMFailure)
	}
	nameVariant, err := oleutil.GetProperty(inboxDispatch, "Name")
	inboxDispatch.Release()
	if err != nil {
		mapiDispatch.Release()
		appDispatch.Release()
		ole.CoUninitialize()
		return fmt.Errorf("%w: MAPI validation failed (Inbox.Name): %v", ErrCOMFailure, err)
	}
	nameVariant.Clear()

	s.ole = appDispatch
	s.mapi = mapiDispatch
	s.connected = true

	return nil
}

func (s *outlookSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// COM Release can panic with an access violation (0xc0000005) when the
	// underlying server has disconnected (e.g. Exchange is unreachable, or
	// Outlook was closed while we held a reference). We recover from this
	// to ensure a clean shutdown instead of crashing the process.
	safeRelease := func(d **ole.IDispatch) {
		if *d == nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				// COM pointer was stale — nothing to release.
			}
		}()
		(*d).Release()
		*d = nil
	}

	safeRelease(&s.mapi)
	safeRelease(&s.ole)

	if s.connected {
		ole.CoUninitialize()
	}

	s.connected = false

	return nil
}

func (s *outlookSession) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.connected
}
