package outlook

import "github.com/jinkp/outlook-go-mcp/internal/domain"

var (
	ErrNotFound      = domain.ErrNotFound
	ErrNotConnected  = domain.ErrNotConnected
	ErrPolicyDenied  = domain.ErrPolicyDenied
	ErrInvalidParams = domain.ErrInvalidParams
	ErrCOMFailure    = domain.ErrCOMFailure
)
