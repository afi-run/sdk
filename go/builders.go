// Package afi — Step builders for the per-adapter calldata format consumed by
// Lib.runRoutes. These functions pack the fields into the exact big-endian byte
// layout expected by each DEX adapter contract.
//
// Use the resulting Step values with EncodeSteps to build the `params` argument
// of Afi.swap / Afi.swapFor.
package afi

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Route IDs registered in RouteRegistry (mirror DeployInfra constants).
const (
	RouteIDCurve128       uint16 = 1
	RouteIDCakeV3         uint16 = 2
	RouteIDUniV3          uint16 = 3
	RouteIDUniV4          uint16 = 4
	RouteIDAerodrome      uint16 = 5
	RouteIDCurve256       uint16 = 6
	RouteIDFluid          uint16 = 7
	RouteIDBalancerV3     uint16 = 9
	RouteIDAaveLiquidator uint16 = 10
)

// validateMinOut128 enforces the uint128 ceiling on a slippage bound.
func validateMinOut128(name string, v *big.Int) error {
	if v == nil {
		return fmt.Errorf("%s: minOut is nil", name)
	}
	if v.Sign() < 0 {
		return fmt.Errorf("%s: minOut is negative", name)
	}
	if v.BitLen() > 128 {
		return fmt.Errorf("%s: minOut exceeds uint128", name)
	}
	return nil
}

// validateUint160 checks a sqrtPriceLimit value fits in 160 bits.
func validateUint160(name string, v *big.Int) error {
	if v == nil {
		return nil // treated as zero
	}
	if v.Sign() < 0 {
		return fmt.Errorf("%s: sqrtPriceLimit is negative", name)
	}
	if v.BitLen() > 160 {
		return fmt.Errorf("%s: sqrtPriceLimit exceeds uint160", name)
	}
	return nil
}

// putUint128BE writes a big.Int as 16 big-endian bytes into buf[off:off+16].
func putUint128BE(buf []byte, off int, v *big.Int) {
	if v == nil {
		return
	}
	b := v.Bytes()
	copy(buf[off+16-len(b):off+16], b)
}

// putUint160BE writes a big.Int as 20 big-endian bytes into buf[off:off+20].
func putUint160BE(buf []byte, off int, v *big.Int) {
	if v == nil {
		return
	}
	b := v.Bytes()
	copy(buf[off+20-len(b):off+20], b)
}

// putUint24BE writes the low 24 bits of v as 3 big-endian bytes.
func putUint24BE(buf []byte, off int, v uint32) {
	buf[off] = byte(v >> 16)
	buf[off+1] = byte(v >> 8)
	buf[off+2] = byte(v)
}

// putInt24BE encodes a signed 24-bit integer (two's-complement) into 3 bytes.
// Returns an error if v is outside [-2^23, 2^23-1].
func putInt24BE(buf []byte, off int, v int32) error {
	const min24 = -(1 << 23)
	const max24 = (1 << 23) - 1
	if v < min24 || v > max24 {
		return fmt.Errorf("int24 out of range: %d", v)
	}
	// Mask to 24 bits — two's-complement representation for negatives.
	u := uint32(v) & 0x00FFFFFF
	putUint24BE(buf, off, u)
	return nil
}

// BuildUniV3Step packs a UniV3 step (routeID=3).
// Layout: tokenOut(20) + fee(3) + amountOutMinimum(16) + sqrtPriceLimitX96(20) = 59 bytes.
func BuildUniV3Step(tokenOut common.Address, fee uint32, minOut *big.Int, sqrtPriceLimitX96 *big.Int) (Step, error) {
	if err := validateMinOut128("BuildUniV3Step", minOut); err != nil {
		return Step{}, err
	}
	if err := validateUint160("BuildUniV3Step", sqrtPriceLimitX96); err != nil {
		return Step{}, err
	}
	if fee >= 1<<24 {
		return Step{}, fmt.Errorf("BuildUniV3Step: fee %d exceeds uint24", fee)
	}
	buf := make([]byte, 59)
	copy(buf[0:20], tokenOut.Bytes())
	putUint24BE(buf, 20, fee)
	putUint128BE(buf, 23, minOut)
	putUint160BE(buf, 39, sqrtPriceLimitX96)
	return Step{ID: RouteIDUniV3, Data: buf}, nil
}

// BuildCakeV3Step packs a PancakeSwap V3 step (routeID=2). Same layout as UniV3.
func BuildCakeV3Step(tokenOut common.Address, fee uint32, minOut *big.Int, sqrtPriceLimitX96 *big.Int) (Step, error) {
	if err := validateMinOut128("BuildCakeV3Step", minOut); err != nil {
		return Step{}, err
	}
	if err := validateUint160("BuildCakeV3Step", sqrtPriceLimitX96); err != nil {
		return Step{}, err
	}
	if fee >= 1<<24 {
		return Step{}, fmt.Errorf("BuildCakeV3Step: fee %d exceeds uint24", fee)
	}
	buf := make([]byte, 59)
	copy(buf[0:20], tokenOut.Bytes())
	putUint24BE(buf, 20, fee)
	putUint128BE(buf, 23, minOut)
	putUint160BE(buf, 39, sqrtPriceLimitX96)
	return Step{ID: RouteIDCakeV3, Data: buf}, nil
}

// BuildUniV4Step packs a UniV4 step (routeID=4).
// Layout: currency0(20) + currency1(20) + fee(3) + tickSpacing(3) + hooks(20)
//   - zeroForOne(1) + minOut(16) = 83 bytes.
//
// tickSpacing is signed (int24).
func BuildUniV4Step(currency0, currency1 common.Address, fee uint32, tickSpacing int32, hooks common.Address, zeroForOne bool, minOut *big.Int) (Step, error) {
	if err := validateMinOut128("BuildUniV4Step", minOut); err != nil {
		return Step{}, err
	}
	if fee >= 1<<24 {
		return Step{}, fmt.Errorf("BuildUniV4Step: fee %d exceeds uint24", fee)
	}
	buf := make([]byte, 83)
	copy(buf[0:20], currency0.Bytes())
	copy(buf[20:40], currency1.Bytes())
	putUint24BE(buf, 40, fee)
	if err := putInt24BE(buf, 43, tickSpacing); err != nil {
		return Step{}, fmt.Errorf("BuildUniV4Step: %w", err)
	}
	copy(buf[46:66], hooks.Bytes())
	if zeroForOne {
		buf[66] = 1
	}
	putUint128BE(buf, 67, minOut)
	return Step{ID: RouteIDUniV4, Data: buf}, nil
}

// BuildAerodromeStep packs an Aerodrome step (routeID=5).
// Layout: tokenOut(20) + tickSpacing(3, int24 signed) + sqrtPriceLimitX96(20)
//   - amountOutMinimum(16) = 59 bytes.
func BuildAerodromeStep(tokenOut common.Address, tickSpacing int32, sqrtLimit, minOut *big.Int) (Step, error) {
	if err := validateMinOut128("BuildAerodromeStep", minOut); err != nil {
		return Step{}, err
	}
	if err := validateUint160("BuildAerodromeStep", sqrtLimit); err != nil {
		return Step{}, err
	}
	buf := make([]byte, 59)
	copy(buf[0:20], tokenOut.Bytes())
	if err := putInt24BE(buf, 20, tickSpacing); err != nil {
		return Step{}, fmt.Errorf("BuildAerodromeStep: %w", err)
	}
	putUint160BE(buf, 23, sqrtLimit)
	putUint128BE(buf, 43, minOut)
	return Step{ID: RouteIDAerodrome, Data: buf}, nil
}

// BuildBalancerV3Step packs a Balancer V3 step (routeID=9).
// Layout: pool(20) + tokenOut(20) + minOut(16) = 56 bytes.
func BuildBalancerV3Step(pool, tokenOut common.Address, minOut *big.Int) (Step, error) {
	if err := validateMinOut128("BuildBalancerV3Step", minOut); err != nil {
		return Step{}, err
	}
	buf := make([]byte, 56)
	copy(buf[0:20], pool.Bytes())
	copy(buf[20:40], tokenOut.Bytes())
	putUint128BE(buf, 40, minOut)
	return Step{ID: RouteIDBalancerV3, Data: buf}, nil
}

// BuildFluidStep packs a Fluid step (routeID=7).
// Layout: pool(20) + swap0to1(1) + tokenOut(20) + minOut(16) = 57 bytes.
func BuildFluidStep(pool common.Address, swap0to1 bool, tokenOut common.Address, minOut *big.Int) (Step, error) {
	if err := validateMinOut128("BuildFluidStep", minOut); err != nil {
		return Step{}, err
	}
	buf := make([]byte, 57)
	copy(buf[0:20], pool.Bytes())
	if swap0to1 {
		buf[20] = 1
	}
	copy(buf[21:41], tokenOut.Bytes())
	putUint128BE(buf, 41, minOut)
	return Step{ID: RouteIDFluid, Data: buf}, nil
}

// BuildCurve128Step packs a Curve (int128-arg) step (routeID=1).
// Layout: i(1) + j(1) + min_dy(16) + pool(20) + tokenOut(20) = 58 bytes.
func BuildCurve128Step(i, j uint8, minDy *big.Int, pool, tokenOut common.Address) (Step, error) {
	if err := validateMinOut128("BuildCurve128Step", minDy); err != nil {
		return Step{}, err
	}
	buf := make([]byte, 58)
	buf[0] = i
	buf[1] = j
	putUint128BE(buf, 2, minDy)
	copy(buf[18:38], pool.Bytes())
	copy(buf[38:58], tokenOut.Bytes())
	return Step{ID: RouteIDCurve128, Data: buf}, nil
}

// BuildCurve256Step packs a Curve (uint256-arg) step (routeID=6). Same layout as Curve128.
func BuildCurve256Step(i, j uint8, minDy *big.Int, pool, tokenOut common.Address) (Step, error) {
	if err := validateMinOut128("BuildCurve256Step", minDy); err != nil {
		return Step{}, err
	}
	buf := make([]byte, 58)
	buf[0] = i
	buf[1] = j
	putUint128BE(buf, 2, minDy)
	copy(buf[18:38], pool.Bytes())
	copy(buf[38:58], tokenOut.Bytes())
	return Step{ID: RouteIDCurve256, Data: buf}, nil
}

// BuildAaveLiquidatorStep packs an Aave liquidation step (routeID=10).
// Layout: pool(20) + user(20) + collateralAsset(20) = 60 bytes.
func BuildAaveLiquidatorStep(pool, user, collateralAsset common.Address) (Step, error) {
	buf := make([]byte, 60)
	copy(buf[0:20], pool.Bytes())
	copy(buf[20:40], user.Bytes())
	copy(buf[40:60], collateralAsset.Bytes())
	return Step{ID: RouteIDAaveLiquidator, Data: buf}, nil
}
