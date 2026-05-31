package afi

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildUniV3Step_ByteLength(t *testing.T) {
	step, err := BuildUniV3Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		3000,
		big.NewInt(1_000_000),
		big.NewInt(0),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 59 {
		t.Errorf("UniV3 expected 59 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDUniV3 {
		t.Errorf("UniV3 id: got %d want %d", step.ID, RouteIDUniV3)
	}
}

func TestBuildCakeV3Step_ByteLength(t *testing.T) {
	step, err := BuildCakeV3Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		500,
		big.NewInt(1),
		big.NewInt(0),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 59 {
		t.Errorf("CakeV3 expected 59 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDCakeV3 {
		t.Errorf("CakeV3 id: got %d want %d", step.ID, RouteIDCakeV3)
	}
}

func TestBuildUniV4Step_ByteLength(t *testing.T) {
	step, err := BuildUniV4Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		3000,
		60,
		common.Address{},
		true,
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 83 {
		t.Errorf("UniV4 expected 83 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDUniV4 {
		t.Errorf("UniV4 id: got %d want %d", step.ID, RouteIDUniV4)
	}
	// zeroForOne flag is at offset 66.
	if step.Data[66] != 1 {
		t.Errorf("UniV4 zeroForOne byte: got %d want 1", step.Data[66])
	}
}

func TestBuildAerodromeStep_ByteLength(t *testing.T) {
	step, err := BuildAerodromeStep(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		200,
		big.NewInt(0),
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 59 {
		t.Errorf("Aerodrome expected 59 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDAerodrome {
		t.Errorf("Aerodrome id: got %d want %d", step.ID, RouteIDAerodrome)
	}
}

func TestBuildAerodromeStep_NegativeTickSpacing(t *testing.T) {
	step, err := BuildAerodromeStep(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		-1, // sign-extend to 0xFFFFFF
		big.NewInt(0),
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// tickSpacing lives at offset 20..22.
	wantTick := []byte{0xFF, 0xFF, 0xFF}
	if !bytes.Equal(step.Data[20:23], wantTick) {
		t.Errorf("tickSpacing two's-complement: got %x want %x", step.Data[20:23], wantTick)
	}

	step2, err := BuildAerodromeStep(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		-2, // -> 0xFFFFFE
		big.NewInt(0),
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	wantTick2 := []byte{0xFF, 0xFF, 0xFE}
	if !bytes.Equal(step2.Data[20:23], wantTick2) {
		t.Errorf("tickSpacing -2 two's-complement: got %x want %x", step2.Data[20:23], wantTick2)
	}
}

func TestBuildBalancerV3Step_ByteLength(t *testing.T) {
	step, err := BuildBalancerV3Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 56 {
		t.Errorf("BalancerV3 expected 56 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDBalancerV3 {
		t.Errorf("BalancerV3 id: got %d want %d", step.ID, RouteIDBalancerV3)
	}
}

func TestBuildFluidStep_ByteLength(t *testing.T) {
	step, err := BuildFluidStep(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		true,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(1),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 57 {
		t.Errorf("Fluid expected 57 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDFluid {
		t.Errorf("Fluid id: got %d want %d", step.ID, RouteIDFluid)
	}
	if step.Data[20] != 1 {
		t.Errorf("Fluid swap0to1 byte: got %d want 1", step.Data[20])
	}
}

func TestBuildCurve128Step_ByteLength(t *testing.T) {
	step, err := BuildCurve128Step(
		0, 1,
		big.NewInt(1),
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 58 {
		t.Errorf("Curve128 expected 58 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDCurve128 {
		t.Errorf("Curve128 id: got %d want %d", step.ID, RouteIDCurve128)
	}
}

func TestBuildCurve256Step_ByteLength(t *testing.T) {
	step, err := BuildCurve256Step(
		1, 0,
		big.NewInt(1),
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 58 {
		t.Errorf("Curve256 expected 58 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDCurve256 {
		t.Errorf("Curve256 id: got %d want %d", step.ID, RouteIDCurve256)
	}
}

func TestBuildAaveLiquidatorStep_ByteLength(t *testing.T) {
	step, err := BuildAaveLiquidatorStep(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(step.Data) != 60 {
		t.Errorf("AaveLiquidator expected 60 bytes, got %d", len(step.Data))
	}
	if step.ID != RouteIDAaveLiquidator {
		t.Errorf("AaveLiquidator id: got %d want %d", step.ID, RouteIDAaveLiquidator)
	}
}

func TestBuildUniV3Step_MinOutTooBig(t *testing.T) {
	// 2^128 = 1 << 128
	tooBig := new(big.Int).Lsh(big.NewInt(1), 128)
	_, err := BuildUniV3Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		3000,
		tooBig,
		nil,
	)
	if err == nil {
		t.Error("expected error for minOut > uint128, got nil")
	}
}

func TestBuildBalancerV3Step_NilMinOut(t *testing.T) {
	_, err := BuildBalancerV3Step(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		nil,
	)
	if err == nil {
		t.Error("expected error for nil minOut, got nil")
	}
}
