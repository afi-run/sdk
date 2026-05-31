package afi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ─── M.1 Encoder ──────────────────────────────────────────────────────────────

// TestEncodeAcceptOwnership_Selector verifies the function selector matches the
// canonical ABI hash for `acceptOwnership()` — 0x79ba5097. Any drift in the
// minimal Ownable2Step ABI would change this.
func TestEncodeAcceptOwnership_Selector(t *testing.T) {
	data, err := EncodeAcceptOwnership()
	if err != nil {
		t.Fatalf("EncodeAcceptOwnership: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("expected exactly 4 bytes (selector only), got %d", len(data))
	}
	got := fmt.Sprintf("0x%x", data)
	const want = "0x79ba5097" // keccak256("acceptOwnership()")[:4]
	if got != want {
		t.Errorf("selector = %s, want %s", got, want)
	}
}

// ─── L.4 SweepNMRProfit input validation ─────────────────────────────────────

// TestSweepNMRProfit_RejectsZeroAmount confirms the SDK refuses to submit a
// sweep for zero (would always revert on-chain).
func TestSweepNMRProfit_RejectsZeroAmount(t *testing.T) {
	srv := newRPCServer(t, rpcHandlers{chainID: 8453})
	defer srv.Close()

	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()

	// Inject Base as the resolved network so nmrAddressForCtx returns a non-zero address.
	_, err := c.SweepNMRProfit(testCtx(), common.HexToAddress("0xdeadbeef"), big.NewInt(0))
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
	if !strings.Contains(err.Error(), "amount must be > 0") {
		t.Errorf("expected amount > 0 error, got: %v", err)
	}
}

// TestSweepNMRProfit_RejectsInsufficientBalance verifies the balance precheck
// catches the case where the NMR contract holds less than the requested sweep.
func TestSweepNMRProfit_RejectsInsufficientBalance(t *testing.T) {
	// chainId for Base, balanceOf returns 100.
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		ethCall: func(to common.Address, data []byte) []byte {
			return encodeUint256(big.NewInt(100))
		},
	})
	defer srv.Close()

	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()

	_, err := c.SweepNMRProfit(testCtx(), common.HexToAddress("0xdeadbeef"), big.NewInt(1000))
	if err == nil {
		t.Fatal("expected insufficient balance error, got nil")
	}
	if !strings.Contains(err.Error(), "holds 100") {
		t.Errorf("expected balance error message, got: %v", err)
	}
}

// ─── L.5 Typed HTTP errors ────────────────────────────────────────────────────

func TestDoHTTPRequest_NetworkError(t *testing.T) {
	// Server that closes the connection immediately.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("hijack not supported")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}))
	defer srv.Close()

	_, err := doHTTPRequest(context.Background(), http.MethodPost, srv.URL, []byte("{}"))
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Errorf("expected *NetworkError, got %T: %v", err, err)
	}
}

func TestDoHTTPRequest_BadRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"missing tokenIn"}`))
	}))
	defer srv.Close()

	_, err := doHTTPRequest(context.Background(), http.MethodPost, srv.URL, []byte("{}"))
	if err == nil {
		t.Fatal("expected BadRequestError, got nil")
	}
	var bad *BadRequestError
	if !errors.As(err, &bad) {
		t.Fatalf("expected *BadRequestError, got %T: %v", err, err)
	}
	if bad.Status != 400 {
		t.Errorf("Status = %d, want 400", bad.Status)
	}
	if !strings.Contains(bad.Body, "missing tokenIn") {
		t.Errorf("Body = %q, want it to contain the message", bad.Body)
	}
}

func TestDoHTTPRequest_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`gateway down`))
	}))
	defer srv.Close()

	_, err := doHTTPRequest(context.Background(), http.MethodPost, srv.URL, []byte("{}"))
	if err == nil {
		t.Fatal("expected ServerError, got nil")
	}
	var srv5xx *ServerError
	if !errors.As(err, &srv5xx) {
		t.Fatalf("expected *ServerError, got %T: %v", err, err)
	}
	if srv5xx.Status != 503 {
		t.Errorf("Status = %d, want 503", srv5xx.Status)
	}
}

// ─── M.2 Batch accept happy path ─────────────────────────────────────────────

// TestAcceptOwnershipBatch_HappyPath runs the full batch flow against a mocked
// JSON-RPC backend that responds to chainId / nonce / gas / fee / send / receipt
// so each AcceptOwnership tx returns a successful receipt.
func TestAcceptOwnershipBatch_HappyPath(t *testing.T) {
	var sent int64

	srv := newRPCServer(t, rpcHandlers{
		chainID:         8453,
		nonce:           42,
		ethCall:         func(common.Address, []byte) []byte { return nil },
		gasEstimate:     500_000,
		baseFee:         big.NewInt(1_000_000_000),
		gasTip:          big.NewInt(1_000_000),
		onSendRaw:       func() { atomic.AddInt64(&sent, 1) },
		receiptSucceeds: true,
	})
	defer srv.Close()

	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()

	contracts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	receipts, err := c.AcceptOwnershipBatch(testCtx(), contracts)
	if err != nil {
		t.Fatalf("AcceptOwnershipBatch: %v", err)
	}
	if len(receipts) != 3 {
		t.Errorf("got %d receipts, want 3", len(receipts))
	}
	if got := atomic.LoadInt64(&sent); got != 3 {
		t.Errorf("eth_sendRawTransaction calls = %d, want 3", got)
	}
}

func TestAcceptOwnershipBatch_StopsAtFirstFailure(t *testing.T) {
	// gasEstimate returns an error after the first call.
	var calls int64
	srv := newRPCServer(t, rpcHandlers{
		chainID: 8453,
		nonce:   1,
		gasEstimateFn: func() (uint64, string) {
			n := atomic.AddInt64(&calls, 1)
			if n == 1 {
				return 200_000, ""
			}
			return 0, "execution reverted"
		},
		baseFee:         big.NewInt(1_000_000_000),
		gasTip:          big.NewInt(1_000_000),
		ethCall:         func(common.Address, []byte) []byte { return nil },
		receiptSucceeds: true,
	})
	defer srv.Close()

	c := mustClientWith(t, srv.URL, testPrivKey)
	defer c.Close()

	contracts := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}
	receipts, err := c.AcceptOwnershipBatch(testCtx(), contracts)
	if err == nil {
		t.Fatal("expected error from second contract, got nil")
	}
	if len(receipts) != 1 {
		t.Errorf("got %d receipts on failure, want 1 (the partial batch)", len(receipts))
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

const testPrivKey = "1111111111111111111111111111111111111111111111111111111111111111"

func testCtx() context.Context { return context.Background() }

func mustClientWith(t *testing.T, rpcURL, privKey string) *Client {
	t.Helper()
	c, err := NewClient(Config{RPCURL: rpcURL, PrivateKey: privKey})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// rpcHandlers is a minimal config struct describing how the fake RPC should
// reply to each method that the SDK might call during these tests.
type rpcHandlers struct {
	chainID         int64
	nonce           uint64
	gasEstimate     uint64
	gasEstimateFn   func() (uint64, string) // optional; overrides gasEstimate
	baseFee         *big.Int
	gasTip          *big.Int
	ethCall         func(to common.Address, data []byte) []byte
	onSendRaw       func()
	receiptSucceeds bool
}

// newRPCServer spins up an httptest.Server that speaks just enough JSON-RPC to
// satisfy the SDK's write path. Unhandled methods return a generic null result.
func newRPCServer(t *testing.T, h rpcHandlers) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(body, &req)

		write := func(result interface{}) {
			resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}
		writeErr := func(msg string) {
			resp := map[string]interface{}{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]interface{}{"code": -32000, "message": msg},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}

		switch req.Method {
		case "eth_chainId":
			write(hexU64(uint64(h.chainID)))
		case "net_version":
			write(fmt.Sprintf("%d", h.chainID))
		case "eth_getTransactionCount":
			write(hexU64(h.nonce))
		case "eth_call":
			// params: [{to, data}, blockTag]
			var p []json.RawMessage
			_ = json.Unmarshal(req.Params, &p)
			var arg struct {
				To   string `json:"to"`
				Data string `json:"data"`
			}
			if len(p) > 0 {
				_ = json.Unmarshal(p[0], &arg)
			}
			var out []byte
			if h.ethCall != nil {
				out = h.ethCall(common.HexToAddress(arg.To), common.FromHex(arg.Data))
			}
			if out == nil {
				out = make([]byte, 32) // zero-padded uint256
			}
			write(hexBytes(out))
		case "eth_estimateGas":
			if h.gasEstimateFn != nil {
				g, e := h.gasEstimateFn()
				if e != "" {
					writeErr(e)
					return
				}
				write(hexU64(g))
				return
			}
			write(hexU64(h.gasEstimate))
		case "eth_maxPriorityFeePerGas":
			write(hexBig(h.gasTip))
		case "eth_gasPrice":
			write(hexBig(h.gasTip))
		case "eth_getBlockByNumber":
			// Need a header with baseFeePerGas + number.
			write(map[string]interface{}{
				"number":           "0x1",
				"hash":             "0x" + strings.Repeat("00", 32),
				"parentHash":       "0x" + strings.Repeat("00", 32),
				"sha3Uncles":       "0x" + strings.Repeat("00", 32),
				"stateRoot":        "0x" + strings.Repeat("00", 32),
				"transactionsRoot": "0x" + strings.Repeat("00", 32),
				"receiptsRoot":     "0x" + strings.Repeat("00", 32),
				"logsBloom":        "0x" + strings.Repeat("00", 256),
				"miner":            "0x" + strings.Repeat("00", 20),
				"difficulty":       "0x0",
				"extraData":        "0x",
				"size":             "0x0",
				"gasLimit":         "0x1c9c380",
				"gasUsed":          "0x0",
				"timestamp":        "0x0",
				"baseFeePerGas":    hexBig(h.baseFee),
			})
		case "eth_sendRawTransaction":
			if h.onSendRaw != nil {
				h.onSendRaw()
			}
			// Return a synthetic tx hash.
			write("0x" + strings.Repeat("ab", 32))
		case "eth_getTransactionReceipt":
			if !h.receiptSucceeds {
				write(nil)
				return
			}
			write(map[string]interface{}{
				"transactionHash":   "0x" + strings.Repeat("ab", 32),
				"transactionIndex":  "0x0",
				"blockHash":         "0x" + strings.Repeat("11", 32),
				"blockNumber":       "0x2",
				"from":              "0x" + strings.Repeat("00", 20),
				"to":                "0x" + strings.Repeat("00", 20),
				"cumulativeGasUsed": "0x5208",
				"gasUsed":           "0x5208",
				"contractAddress":   nil,
				"logs":              []interface{}{},
				"logsBloom":         "0x" + strings.Repeat("00", 256),
				"status":            "0x1",
				"effectiveGasPrice": "0x1",
			})
		default:
			write(nil)
		}
	}))
}

func hexU64(n uint64) string { return fmt.Sprintf("0x%x", n) }
func hexBig(b *big.Int) string {
	if b == nil {
		return "0x0"
	}
	return fmt.Sprintf("0x%x", b)
}
func hexBytes(b []byte) string { return "0x" + common.Bytes2Hex(b) }

// encodeUint256 left-pads `v` to 32 bytes for use as eth_call return data.
func encodeUint256(v *big.Int) []byte {
	out := make([]byte, 32)
	v.FillBytes(out)
	return out
}
