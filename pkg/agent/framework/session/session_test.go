package session_test

import (
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/session"
)

func TestNewManager(t *testing.T) {
	t.Run("create new manager", func(t *testing.T) {
		mgr := session.NewManager()
		if mgr == nil {
			t.Fatal("NewManager() returned nil")
		}
	})
}

func TestGetUnderlying(t *testing.T) {
	t.Run("get underlying manager", func(t *testing.T) {
		mgr := session.NewManager()
		underlying := mgr.GetUnderlying()
		if underlying == nil {
			t.Error("GetUnderlying() returned nil")
		}
	})
}

func TestManagerLifecycle(t *testing.T) {
	t.Run("manager lifecycle", func(t *testing.T) {
		// Create manager
		mgr := session.NewManager()
		if mgr == nil {
			t.Fatal("Failed to create manager")
		}

		// Verify it has underlying implementation
		underlying := mgr.GetUnderlying()
		if underlying == nil {
			t.Error("Manager has no underlying implementation")
		}

		// Create multiple managers
		mgr2 := session.NewManager()
		if mgr2 == nil {
			t.Fatal("Failed to create second manager")
		}

		// Verify they are independent
		if mgr == mgr2 {
			t.Error("Managers should be independent instances")
		}
	})
}

func TestManagerConcurrency(t *testing.T) {
	t.Run("concurrent manager creation", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan *session.Manager, numGoroutines)

		// Create managers concurrently
		for i := 0; i < numGoroutines; i++ {
			go func() {
				mgr := session.NewManager()
				results <- mgr
			}()
		}

		// Collect results
		managers := make([]*session.Manager, 0, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			mgr := <-results
			if mgr == nil {
				t.Error("Concurrent creation returned nil manager")
			}
			managers = append(managers, mgr)
		}

		// Verify all managers are valid
		for i, mgr := range managers {
			if mgr.GetUnderlying() == nil {
				t.Errorf("Manager %d has no underlying implementation", i)
			}
		}
	})
}

// Benchmark tests
func BenchmarkNewManager(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = session.NewManager()
	}
}

func BenchmarkGetUnderlying(b *testing.B) {
	mgr := session.NewManager()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.GetUnderlying()
	}
}
