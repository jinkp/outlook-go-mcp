//go:build windows

package outlook

import (
	"fmt"
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

// PingOutlook performs a full COM lifecycle on the CURRENT OS thread:
// CoInitializeEx → GetActiveObject/CreateObject → GetNamespace("MAPI")
// → GetDefaultFolder(Inbox) → read .Name → clean up and return.
//
// The caller MUST have called runtime.LockOSThread() before calling this.
// This function is designed for the _ping subprocess, where COM runs on
// the main goroutine (which is the main OS thread).
func PingOutlook() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return fmt.Errorf("%w: initialize COM: %v", ErrCOMFailure, err)
	}
	defer ole.CoUninitialize()

	appUnknown, err := oleutil.GetActiveObject("Outlook.Application")
	if err != nil {
		appUnknown, err = oleutil.CreateObject("Outlook.Application")
		if err != nil {
			return fmt.Errorf("%w: connect to Outlook: %v", ErrNotConnected, err)
		}
	}
	defer appUnknown.Release()

	disp, err := appUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("%w: query Outlook dispatch: %v", ErrCOMFailure, err)
	}
	defer disp.Release()

	nsVariant, err := oleutil.CallMethod(disp, "GetNamespace", "MAPI")
	if err != nil {
		return fmt.Errorf("%w: get MAPI namespace: %v", ErrCOMFailure, err)
	}
	ns := nsVariant.ToIDispatch()
	if ns == nil {
		return fmt.Errorf("%w: MAPI namespace is nil", ErrCOMFailure)
	}
	defer ns.Release()

	inboxVariant, err := oleutil.CallMethod(ns, "GetDefaultFolder", 6) // olFolderInbox
	if err != nil {
		return fmt.Errorf("%w: get inbox folder: %v", ErrCOMFailure, err)
	}
	inbox := inboxVariant.ToIDispatch()
	if inbox == nil {
		return fmt.Errorf("%w: inbox folder is nil", ErrCOMFailure)
	}
	defer inbox.Release()

	nameVar, err := oleutil.GetProperty(inbox, "Name")
	if err != nil {
		return fmt.Errorf("%w: get inbox name: %v", ErrCOMFailure, err)
	}
	defer nameVar.Clear()

	if nameVar.ToString() == "" {
		return fmt.Errorf("%w: inbox name is empty", ErrCOMFailure)
	}

	return nil
}

// InitCOM initializes the COM apartment on the current OS thread. This MUST
// be called once from the thread that will make all COM calls (i.e. the
// executor worker goroutine after runtime.LockOSThread). Connect() assumes
// COM is already initialized.
func (s *outlookSession) InitCOM() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return fmt.Errorf("%w: initialize COM apartment: %v", ErrCOMFailure, err)
	}
	return nil
}

// UninitCOM tears down the COM apartment. Called once when the executor stops.
func (s *outlookSession) UninitCOM() {
	ole.CoUninitialize()
}

func (s *outlookSession) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	appUnknown, err := oleutil.GetActiveObject("Outlook.Application")
	if err != nil {
		appUnknown, err = oleutil.CreateObject("Outlook.Application")
		if err != nil {
			return fmt.Errorf("%w: connect to Outlook application: %v", ErrNotConnected, err)
		}
	}
	defer appUnknown.Release()

	appDispatch, err := appUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("%w: query Outlook dispatch: %v", ErrCOMFailure, err)
	}

	mapiVariant, err := oleutil.CallMethod(appDispatch, "GetNamespace", "MAPI")
	if err != nil {
		appDispatch.Release()
		return fmt.Errorf("%w: get MAPI namespace: %v", ErrCOMFailure, err)
	}
	defer mapiVariant.Clear()

	mapiDispatch := mapiVariant.ToIDispatch()
	if mapiDispatch == nil {
		appDispatch.Release()
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
		return fmt.Errorf("%w: MAPI validation failed (GetDefaultFolder): %v", ErrCOMFailure, err)
	}
	inboxDispatch := inboxVariant.ToIDispatch()
	if inboxDispatch == nil {
		mapiDispatch.Release()
		appDispatch.Release()
		return fmt.Errorf("%w: MAPI validation failed: GetDefaultFolder returned nil", ErrCOMFailure)
	}
	nameVariant, err := oleutil.GetProperty(inboxDispatch, "Name")
	inboxDispatch.Release()
	if err != nil {
		mapiDispatch.Release()
		appDispatch.Release()
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

	// NOTE: CoUninitialize is NOT called here. The COM apartment lifecycle
	// is managed by the COMExecutor (InitCOM at start, UninitCOM at stop).
	s.connected = false

	return nil
}

func (s *outlookSession) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.connected
}
