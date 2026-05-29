package afi

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPendingTxWait(t *testing.T) {
	t.Run("returns receipt from waitFn", func(t *testing.T) {
		want := &TxReceipt{BlockNumber: 42, GasUsed: 21000}
		p := &PendingTx{
			TxHash: "0xabc",
			waitFn: func(ctx context.Context) (*TxReceipt, error) {
				return want, nil
			},
		}
		got, err := p.Wait(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("propagates error from waitFn", func(t *testing.T) {
		boom := errors.New("boom")
		p := &PendingTx{
			TxHash: "0xabc",
			waitFn: func(ctx context.Context) (*TxReceipt, error) {
				return nil, boom
			},
		}
		_, err := p.Wait(context.Background())
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})

	t.Run("forwards context to waitFn", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var received context.Context
		p := &PendingTx{
			waitFn: func(c context.Context) (*TxReceipt, error) {
				received = c
				return &TxReceipt{}, nil
			},
		}
		p.Wait(ctx)
		if received == nil || received.Err() == nil {
			t.Error("waitFn did not receive the cancelled context")
		}
	})
}

func TestPendingSwapWait(t *testing.T) {
	t.Run("returns SwapResult from waitFn", func(t *testing.T) {
		want := &SwapResult{
			TxHash:      common.HexToHash("0xdeadbeef"),
			BlockNumber: 100,
			AmountIn:    big.NewInt(1000),
			AmountOut:   big.NewInt(2000),
			TokenIn:     common.HexToAddress("0x1"),
			TokenOut:    common.HexToAddress("0x2"),
			GasUsed:     50000,
		}
		p := &PendingSwap{
			TxHash: "0xabc",
			waitFn: func(ctx context.Context) (*SwapResult, error) {
				return want, nil
			},
		}
		got, err := p.Wait(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("propagates error from waitFn", func(t *testing.T) {
		boom := errors.New("reverted")
		p := &PendingSwap{
			waitFn: func(ctx context.Context) (*SwapResult, error) {
				return nil, boom
			},
		}
		_, err := p.Wait(context.Background())
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})

	t.Run("forwards context to waitFn", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var received context.Context
		p := &PendingSwap{
			waitFn: func(c context.Context) (*SwapResult, error) {
				received = c
				return &SwapResult{}, nil
			},
		}
		p.Wait(ctx)
		if received == nil || received.Err() == nil {
			t.Error("waitFn did not receive the cancelled context")
		}
	})
}
