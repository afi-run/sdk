package afi

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestLogger_FiresOnLoggedCalls(t *testing.T) {
	var calls atomic.Int32
	var captured LogEvent

	c, _ := NewClient(Config{
		RPCURL: "http://localhost:1",
		Logger: func(e LogEvent) {
			calls.Add(1)
			captured = e
		},
	})
	defer c.Close()

	err := c.logged("dummy", func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 logger call, got %d", calls.Load())
	}
	if captured.Method != "dummy" || !captured.OK {
		t.Errorf("captured event mismatch: %+v", captured)
	}
}

func TestLogger_CapturesError(t *testing.T) {
	var captured LogEvent

	c, _ := NewClient(Config{
		RPCURL: "http://localhost:1",
		Logger: func(e LogEvent) { captured = e },
	})
	defer c.Close()

	boom := errors.New("boom")
	err := c.logged("failing", func() error { return boom })
	if !errors.Is(err, boom) {
		t.Errorf("expected boom propagation, got %v", err)
	}
	if captured.OK {
		t.Error("expected OK=false on failure")
	}
	if !errors.Is(captured.Err, boom) {
		t.Errorf("captured Err mismatch: %v", captured.Err)
	}
}

func TestLogger_NilLoggerSkipsEmission(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	// Should NOT panic even though no logger is set
	err := c.logged("dummy", func() error { return nil })
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
}

func TestSetLogger_ReplacesLogger(t *testing.T) {
	count1 := 0
	c, _ := NewClient(Config{
		RPCURL: "http://localhost:1",
		Logger: func(_ LogEvent) { count1++ },
	})
	defer c.Close()

	_ = c.logged("a", func() error { return nil })

	count2 := 0
	c.SetLogger(func(_ LogEvent) { count2++ })
	_ = c.logged("b", func() error { return nil })

	if count1 != 1 {
		t.Errorf("first logger count = %d, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("second logger count = %d, want 1", count2)
	}
}

func TestWaitForTxOptions_DefaultsApplied(t *testing.T) {
	c, _ := NewClient(Config{RPCURL: "http://localhost:1"})
	defer c.Close()

	// WaitForTx will block on the RPC — we just want to exercise the option-merge logic via cancel.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.WaitForTx(ctx, "0x0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected ctx.Err() when context is pre-cancelled")
	}
}
