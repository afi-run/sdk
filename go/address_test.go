package afi

import "testing"

func TestIsAddress(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0xB8cC65321d169D55b93b4402D795701c6B308ce4", true},
		{"0x0000000000000000000000000000000000000000", true},
		{"0xshort", false},
		{"notanaddress", false},
		{"B8cC65321d169D55b93b4402D795701c6B308ce4", false}, // no 0x
		{"0xZ8cC65321d169D55b93b4402D795701c6B308ce4", false},
	}
	for _, c := range cases {
		if got := IsAddress(c.in); got != c.want {
			t.Errorf("IsAddress(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestChecksum_ReturnsEIP55(t *testing.T) {
	got := Checksum("0xb8cc65321d169d55b93b4402d795701c6b308ce4")
	want := "0xB8cC65321d169D55b93b4402D795701c6B308ce4"
	if got != want {
		t.Errorf("Checksum = %q, want %q", got, want)
	}
}

func TestChecksum_IsIdempotent(t *testing.T) {
	a := Checksum("0xb8cc65321d169d55b93b4402d795701c6b308ce4")
	if Checksum(a) != a {
		t.Error("Checksum should be idempotent")
	}
}

func TestChecksum_InvalidReturnsZero(t *testing.T) {
	if Checksum("not-an-address") != ZeroAddressHex {
		t.Error("invalid input should return zero address")
	}
}

func TestIsZeroAddress(t *testing.T) {
	if !IsZeroAddress("0x0000000000000000000000000000000000000000") {
		t.Error("expected true for canonical zero")
	}
	if !IsZeroAddress("0X0000000000000000000000000000000000000000") {
		t.Error("expected case-insensitive match")
	}
	if IsZeroAddress("0xB8cC65321d169D55b93b4402D795701c6B308ce4") {
		t.Error("expected false for non-zero")
	}
}

func TestEqualAddresses(t *testing.T) {
	a := "0xB8cC65321d169D55b93b4402D795701c6B308ce4"
	b := "0xb8cc65321d169d55b93b4402d795701c6b308ce4"
	if !EqualAddresses(a, b) {
		t.Error("expected case-insensitive equality")
	}
	if EqualAddresses(a, "0x4200000000000000000000000000000000000006") {
		t.Error("expected inequality")
	}
}
