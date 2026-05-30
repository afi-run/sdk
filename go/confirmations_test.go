package afi

import (
	"context"
	"testing"
)

func TestPendingSwap_Wait_ForwardsOptions(t *testing.T) {
	var got WaitForTxOptions

	p := &PendingSwap{
		TxHash: "0xabc",
		waitFn: func(_ context.Context, opts WaitForTxOptions) (*SwapResult, error) {
			got = opts
			return &SwapResult{}, nil
		},
	}

	_, err := p.Wait(context.Background(), WaitForTxOptions{Confirmations: 3, TimeoutMs: 5_000})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.Confirmations != 3 || got.TimeoutMs != 5_000 {
		t.Errorf("opts not forwarded: %+v", got)
	}
}

func TestPendingSwap_Wait_DefaultsWhenNoOpts(t *testing.T) {
	var got WaitForTxOptions
	p := &PendingSwap{
		waitFn: func(_ context.Context, opts WaitForTxOptions) (*SwapResult, error) {
			got = opts
			return &SwapResult{}, nil
		},
	}
	_, _ = p.Wait(context.Background())
	if got.Confirmations != 0 || got.TimeoutMs != 0 {
		t.Errorf("expected zero-value opts, got %+v", got)
	}
}

func TestPendingTx_Wait_ForwardsOptions(t *testing.T) {
	var got WaitForTxOptions

	p := &PendingTx{
		waitFn: func(_ context.Context, opts WaitForTxOptions) (*TxReceipt, error) {
			got = opts
			return &TxReceipt{}, nil
		},
	}
	_, _ = p.Wait(context.Background(), WaitForTxOptions{Confirmations: 2})
	if got.Confirmations != 2 {
		t.Errorf("confirmations not forwarded: %+v", got)
	}
}
