//go:build integration && windows

package outlook

import (
	"context"
	"testing"
)

func newIntegrationRuntime(t *testing.T) (OutlookSession, *COMExecutor, func()) {
	t.Helper()

	session := NewOutlookSession()
	executor := NewCOMExecutor(session)
	if err := executor.Start(); err != nil {
		t.Fatalf("executor.Start() error = %v", err)
	}

	return session, executor, func() {
		executor.Stop()
	}
}

func deleteDraftByID(ctx context.Context, executor *COMExecutor, id string) error {
	return executor.Submit(ctx, func() error {
		// Integration-only exception: cleanup needs direct item deletion after the store creates a real draft.
		// This still runs inside the executor-owned COM thread, so the production isolation rule is preserved.
		session, ok := executor.session.(*outlookSession)
		if !ok || session == nil {
			return ErrNotConnected
		}

		item, err := getMailItemByID(session, id)
		if err != nil {
			return err
		}
		defer item.Release()

		return deleteMailItem(item)
	})
}
