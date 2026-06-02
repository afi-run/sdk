package afi

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// routeRegistryParsedABI is the parsed minimal RouteRegistry ABI.
var routeRegistryParsedABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(RouteRegistryABI))
	if err != nil {
		panic("invalid RouteRegistryABI: " + err.Error())
	}
	return a
}()

// DeploymentReport summarizes a multi-call sanity check against a deployment.
type DeploymentReport struct {
	Ok     bool
	Issues []string
}

// networkForChain finds the Network matching the given chain id.
func networkForChain(chainID int64) (Network, bool) {
	for n, id := range NetworkChainIDs {
		if id == chainID {
			return n, true
		}
	}
	return "", false
}

// addressForChain looks up a deployed address from a per-network map.
// Returns the resolved Network on hit and a friendly error on miss.
func addressForChain(name string, table map[Network]common.Address, chainID int64) (common.Address, Network, error) {
	net, ok := networkForChain(chainID)
	if !ok {
		return common.Address{}, "", fmt.Errorf("%s: unknown chain id %d", name, chainID)
	}
	addr, ok := table[net]
	if !ok || addr == (common.Address{}) {
		return common.Address{}, net, fmt.Errorf("%s: not deployed on %s", name, net)
	}
	return addr, net, nil
}

// callRead packs fn(args...) against parsedABI, executes eth_call to `to`, and
// returns the raw return data ready to be Unpacked.
func (c *Client) callRead(ctx context.Context, to common.Address, parsedABI abi.ABI, fn string, args ...interface{}) ([]byte, error) {
	input, err := parsedABI.Pack(fn, args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", fn, err)
	}
	raw, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &to, Data: input}, nil)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", fn, err)
	}
	return raw, nil
}

// IsPaused reads Afi.paused() on the given chain.
func (c *Client) IsPaused(ctx context.Context, chainID int64) (bool, error) {
	addr, _, err := addressForChain("IsPaused", AfiAddresses, chainID)
	if err != nil {
		return false, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "paused")
	if err != nil {
		return false, err
	}
	out, err := afiParsedABI.Unpack("paused", raw)
	if err != nil {
		return false, err
	}
	return out[0].(bool), nil
}

// GetFeeBpsOf reads Afi.feeBpsOf(user) on the given chain.
func (c *Client) GetFeeBpsOf(ctx context.Context, user common.Address, chainID int64) (uint16, error) {
	addr, _, err := addressForChain("GetFeeBpsOf", AfiAddresses, chainID)
	if err != nil {
		return 0, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "feeBpsOf", user)
	if err != nil {
		return 0, err
	}
	out, err := afiParsedABI.Unpack("feeBpsOf", raw)
	if err != nil {
		return 0, err
	}
	return out[0].(uint16), nil
}

// HasRules reads Afi.hasRules() on the given chain.
func (c *Client) HasRules(ctx context.Context, chainID int64) (bool, error) {
	addr, _, err := addressForChain("HasRules", AfiAddresses, chainID)
	if err != nil {
		return false, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "hasRules")
	if err != nil {
		return false, err
	}
	out, err := afiParsedABI.Unpack("hasRules", raw)
	if err != nil {
		return false, err
	}
	return out[0].(bool), nil
}

// GetTreasuryAddress reads Afi.treasury() on the given chain.
func (c *Client) GetTreasuryAddress(ctx context.Context, chainID int64) (common.Address, error) {
	addr, _, err := addressForChain("GetTreasuryAddress", AfiAddresses, chainID)
	if err != nil {
		return common.Address{}, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "treasury")
	if err != nil {
		return common.Address{}, err
	}
	out, err := afiParsedABI.Unpack("treasury", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// GetRegistryAddress reads Afi.registry() on the given chain.
func (c *Client) GetRegistryAddress(ctx context.Context, chainID int64) (common.Address, error) {
	addr, _, err := addressForChain("GetRegistryAddress", AfiAddresses, chainID)
	if err != nil {
		return common.Address{}, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "registry")
	if err != nil {
		return common.Address{}, err
	}
	out, err := afiParsedABI.Unpack("registry", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// GetPrimaryOperator reads Afi.primaryOperator() on the given chain.
func (c *Client) GetPrimaryOperator(ctx context.Context, chainID int64) (common.Address, error) {
	addr, _, err := addressForChain("GetPrimaryOperator", AfiAddresses, chainID)
	if err != nil {
		return common.Address{}, err
	}
	raw, err := c.callRead(ctx, addr, afiParsedABI, "primaryOperator")
	if err != nil {
		return common.Address{}, err
	}
	out, err := afiParsedABI.Unpack("primaryOperator", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// IsAfiOperator reads Afi.isOperator(addr) on the given chain.
func (c *Client) IsAfiOperator(ctx context.Context, addr common.Address, chainID int64) (bool, error) {
	afiAddr, _, err := addressForChain("IsAfiOperator", AfiAddresses, chainID)
	if err != nil {
		return false, err
	}
	raw, err := c.callRead(ctx, afiAddr, afiParsedABI, "isOperator", addr)
	if err != nil {
		return false, err
	}
	out, err := afiParsedABI.Unpack("isOperator", raw)
	if err != nil {
		return false, err
	}
	return out[0].(bool), nil
}

// GetOwner reads owner() against `contract` using the Afi ABI (Ownable2Step).
func (c *Client) GetOwner(ctx context.Context, contract common.Address, chainID int64) (common.Address, error) {
	_ = chainID // chain id is informational here — caller already has the address.
	raw, err := c.callRead(ctx, contract, afiParsedABI, "owner")
	if err != nil {
		return common.Address{}, err
	}
	out, err := afiParsedABI.Unpack("owner", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// GetPendingOwner reads pendingOwner() against `contract`.
func (c *Client) GetPendingOwner(ctx context.Context, contract common.Address, chainID int64) (common.Address, error) {
	_ = chainID
	raw, err := c.callRead(ctx, contract, afiParsedABI, "pendingOwner")
	if err != nil {
		return common.Address{}, err
	}
	out, err := afiParsedABI.Unpack("pendingOwner", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// GetRoute reads RouteRegistry.getRoute(id) on the given chain.
func (c *Client) GetRoute(ctx context.Context, id uint16, chainID int64) (common.Address, error) {
	reg, err := c.GetRegistryAddress(ctx, chainID)
	if err != nil {
		return common.Address{}, err
	}
	raw, err := c.callRead(ctx, reg, routeRegistryParsedABI, "getRoute", id)
	if err != nil {
		return common.Address{}, err
	}
	out, err := routeRegistryParsedABI.Unpack("getRoute", raw)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// ListRoutes returns the configured RouteRegistry mapping for IDs 1-9. Routes
// that revert (e.g. unset) are silently skipped — only non-zero hits land in
// the map.
func (c *Client) ListRoutes(ctx context.Context, chainID int64) (map[uint16]common.Address, error) {
	reg, err := c.GetRegistryAddress(ctx, chainID)
	if err != nil {
		return nil, err
	}
	out := make(map[uint16]common.Address)
	for id := uint16(1); id <= 9; id++ {
		raw, err := c.callRead(ctx, reg, routeRegistryParsedABI, "getRoute", id)
		if err != nil {
			continue // skip ids that revert
		}
		unp, err := routeRegistryParsedABI.Unpack("getRoute", raw)
		if err != nil {
			continue
		}
		addr := unp[0].(common.Address)
		if addr != (common.Address{}) {
			out[id] = addr
		}
	}
	return out, nil
}

// GetTreasuryBalance reads the ERC20 balance of the Afi treasury for `token`.
func (c *Client) GetTreasuryBalance(ctx context.Context, token common.Address, chainID int64) (*big.Int, error) {
	treasury, err := c.GetTreasuryAddress(ctx, chainID)
	if err != nil {
		return nil, err
	}
	return getBalance(ctx, c.eth, c.erc20, token, treasury)
}

// VerifyDeployment runs a quick sanity check against the deployed Afi contract,
// bundling reads via the on-chain Multicall3 contract. Reports include zero
// addresses, unset treasury, and unset primary operator.
func (c *Client) VerifyDeployment(ctx context.Context, chainID int64) (*DeploymentReport, error) {
	afiAddr, _, err := addressForChain("VerifyDeployment", AfiAddresses, chainID)
	if err != nil {
		return nil, err
	}

	report := &DeploymentReport{Ok: true}

	calls := []Multicall3Call{}
	addCall := func(target common.Address, parsedABI abi.ABI, fn string, args ...interface{}) error {
		data, err := parsedABI.Pack(fn, args...)
		if err != nil {
			return err
		}
		calls = append(calls, Multicall3Call{Target: target, AllowFailure: true, CallData: data})
		return nil
	}
	if err := addCall(afiAddr, afiParsedABI, "treasury"); err != nil {
		return nil, err
	}
	if err := addCall(afiAddr, afiParsedABI, "registry"); err != nil {
		return nil, err
	}
	if err := addCall(afiAddr, afiParsedABI, "owner"); err != nil {
		return nil, err
	}
	if err := addCall(afiAddr, afiParsedABI, "primaryOperator"); err != nil {
		return nil, err
	}
	if err := addCall(afiAddr, afiParsedABI, "paused"); err != nil {
		return nil, err
	}
	if err := addCall(afiAddr, afiParsedABI, "pendingOwner"); err != nil {
		return nil, err
	}

	results, err := aggregate3(ctx, c.eth, c.multicall3, calls)
	if err != nil {
		return nil, err
	}

	addIssue := func(s string) {
		report.Ok = false
		report.Issues = append(report.Issues, s)
	}
	unpackAddr := func(label string, r Multicall3Result, fn string) common.Address {
		if !r.Success {
			addIssue(fmt.Sprintf("%s call failed", label))
			return common.Address{}
		}
		out, err := afiParsedABI.Unpack(fn, r.ReturnData)
		if err != nil {
			addIssue(fmt.Sprintf("%s decode failed: %v", label, err))
			return common.Address{}
		}
		return out[0].(common.Address)
	}

	if treasury := unpackAddr("treasury", results[0], "treasury"); treasury == (common.Address{}) {
		addIssue("treasury is zero address")
	}
	if registry := unpackAddr("registry", results[1], "registry"); registry == (common.Address{}) {
		addIssue("registry is zero address")
	}
	if owner := unpackAddr("owner", results[2], "owner"); owner == (common.Address{}) {
		addIssue("owner is zero address")
	}
	if primary := unpackAddr("primaryOperator", results[3], "primaryOperator"); primary == (common.Address{}) {
		addIssue("primaryOperator is zero address")
	}
	if results[4].Success {
		out, err := afiParsedABI.Unpack("paused", results[4].ReturnData)
		if err == nil && out[0].(bool) {
			addIssue("contract is paused")
		}
	}
	// pendingOwner should be zero after a clean handover — anything else means
	// the new owner has not yet called acceptOwnership.
	if results[5].Success {
		if out, err := afiParsedABI.Unpack("pendingOwner", results[5].ReturnData); err == nil {
			if pending := out[0].(common.Address); pending != (common.Address{}) {
				addIssue(fmt.Sprintf("Afi pendingOwner is %s — handover not complete (call acceptOwnership)", pending.Hex()))
			}
		}
	}

	return report, nil
}
