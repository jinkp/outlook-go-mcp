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

	s.ole = appDispatch
	s.mapi = mapiDispatch
	s.connected = true

	return nil
}

func (s *outlookSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mapi != nil {
		s.mapi.Release()
		s.mapi = nil
	}

	if s.ole != nil {
		s.ole.Release()
		s.ole = nil
	}

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
