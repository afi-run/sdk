package afi

import (
	"bytes"
	"testing"
)

func TestEncodeSteps_Empty(t *testing.T) {
	got, err := EncodeSteps(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []byte{0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("empty input: got %x, want %x", got, want)
	}
}

func TestEncodeSteps_SingleStep(t *testing.T) {
	got, err := EncodeSteps([]Step{{ID: 3, Data: []byte{0x01, 0x02}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []byte{0x01, 0x00, 0x03, 0x00, 0x02, 0x01, 0x02}
	if !bytes.Equal(got, want) {
		t.Errorf("single step: got %x, want %x", got, want)
	}
}

func TestEncodeSteps_TooManySteps(t *testing.T) {
	steps := make([]Step, 256)
	if _, err := EncodeSteps(steps); err == nil {
		t.Error("expected error for >255 steps, got nil")
	}
}

func TestEncodeSteps_DataTooLarge(t *testing.T) {
	steps := []Step{{ID: 1, Data: make([]byte, 65536)}}
	if _, err := EncodeSteps(steps); err == nil {
		t.Error("expected error for data >65535, got nil")
	}
}
