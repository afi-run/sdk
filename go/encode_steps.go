package afi

import (
	"encoding/binary"
	"fmt"
)

// Step is a single hop in a route — DEX adapter id + opaque calldata.
type Step struct {
	ID   uint16
	Data []byte
}

// EncodeSteps packs steps into the tight format consumed by Lib.runRoutes:
//
//	uint8 numSteps + [uint16 routeID | uint16 dataLen | bytes data]×N
//
// Returns an error on >255 steps or data >65535 bytes per step (cannot be
// represented in u8/u16). Empty input encodes to a single 0x00 byte; callers
// that want a true no-op should pass nil bytes directly to Afi.swap.
func EncodeSteps(steps []Step) ([]byte, error) {
	if len(steps) > 255 {
		return nil, fmt.Errorf("EncodeSteps: %d steps exceeds u8 max", len(steps))
	}
	total := 1
	for _, s := range steps {
		if len(s.Data) > 65535 {
			return nil, fmt.Errorf("EncodeSteps: step data %d exceeds u16 max", len(s.Data))
		}
		total += 4 + len(s.Data)
	}
	out := make([]byte, 0, total)
	out = append(out, byte(len(steps)))
	for _, s := range steps {
		var idBuf [2]byte
		binary.BigEndian.PutUint16(idBuf[:], s.ID)
		out = append(out, idBuf[:]...)
		var lenBuf [2]byte
		binary.BigEndian.PutUint16(lenBuf[:], uint16(len(s.Data)))
		out = append(out, lenBuf[:]...)
		out = append(out, s.Data...)
	}
	return out, nil
}
