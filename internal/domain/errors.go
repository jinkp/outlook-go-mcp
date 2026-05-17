package domain

import "errors"

var (
	ErrNotFound      = errors.New("outlook: item not found")
	ErrNotConnected  = errors.New("outlook: not connected to Outlook")
	ErrPolicyDenied  = errors.New("outlook: action denied by security policy")
	ErrInvalidParams = errors.New("outlook: invalid parameters")
	ErrCOMFailure    = errors.New("outlook: COM automation failure")
)
