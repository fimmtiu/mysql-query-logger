package main

import (
	"fmt"
	"time"
)

const (
	Request  = iota
	Response = iota
)

// Note that the "packet type" byte (OK/EOF/etc.) is technically part of
// the payload, not the header.
const packetHeaderSize = 4

// MySQL constants for the tiny subset of commands that we care about.
const comQuery uint8 = 0x03
const comStmtPrepare uint8 = 0x16
const comStmtExecute uint8 = 0x17

const errorHeader uint8 = 0xFF

type Packet struct {
	Length       uint32
	SequenceId   uint8
	Partial      bool
	Type         uint8
	ConnectionId uint64
	Timestamp    time.Time
	Payload      []byte
}

func NewPacket(t *Traffic) *Packet {
	var offset uint64 = 0
	var p Packet

	if len(t.Data) < packetHeaderSize {
		return nil
	}
	t.Dump()

	// If the traffic is fragmented and doesn't contain the complete packet data, bail early.
	p.Length, offset = fixedInt3(t.Data, offset)
	if len(t.Data) < int(packetHeaderSize+p.Length) {
		output.Debug("skipping packet whose length (%d) is less than the available data (%d)", int(packetHeaderSize+p.Length), len(t.Data))
		if p.Length > 100000 {
			panic("woop woop")
		}
		return nil
	}

	p.SequenceId, offset = fixedInt1(t.Data, offset)
	p.Partial = (p.Length == 0xFFFFFF)
	p.Type = packetType(t)
	p.ConnectionId = t.ConnectionId
	p.Timestamp = t.Timestamp

	packetData := t.ShiftBytes(offset + uint64(p.Length))
	p.Payload = packetData[offset:]
	return &p
}

func (p *Packet) Command() uint8 {
	return p.Payload[0]
}

func (p *Packet) IsRelevantCommand() bool {
	if len(p.Payload) == 0 {
		return false
	}
	return p.Command() == comQuery || p.Command() == comStmtPrepare || p.Command() == comStmtExecute
}

func (p *Packet) ContainsQuery() bool {
	return p.Type == Request && (p.Command() == comQuery || p.Command() == comStmtPrepare)
}

func (p *Packet) ExecutesQuery() bool {
	return p.Type == Request && (p.Command() == comQuery || p.Command() == comStmtExecute)
}

func (p *Packet) IsResponse() bool {
	return p.Type == Response
}

func (p *Packet) IsErrorResponse() bool {
	return p.IsResponse() && len(p.Payload) > 0 && p.Payload[0] == errorHeader
}

// Returns the SQL statement from COM_QUERY and COM_STMT_PREPARE commands.
func (p *Packet) Statement() string {
	if p.ContainsQuery() {
		return string(p.Payload[1:])
	} else {
		return fmt.Sprintf("FIXME: Not a query! (packet type %x)", p.Command())
	}
}

func packetType(t *Traffic) uint8 {
	if t.IsIncoming() {
		return Request
	} else {
		return Response
	}
}

func (p *Packet) packetTypeStr() string {
	if p.IsResponse() {
		return "response"
	} else {
		return "request"
	}
}

// These functions take a stream of data and an offset into that stream,
// and return the data read and the new offset into the stream, advanced by
// the number of bytes it read.

func fixedInt1(data []byte, offset uint64) (uint8, uint64) {
	return data[offset], (offset + 1)
}

func fixedInt2(data []byte, offset uint64) (uint16, uint64) {
	var fixedInt uint16 = uint16(data[offset+1])<<8 | uint16(data[offset])
	return fixedInt, (offset + 2)
}

func fixedInt3(data []byte, offset uint64) (uint32, uint64) {
	var fixedInt uint32 = uint32(data[offset+2])<<16 | uint32(data[offset+1])<<8 | uint32(data[offset])
	return fixedInt, (offset + 3)
}

// And of course I can't line this up nicely, because gofmt.
func fixedInt8(data []byte, offset uint64) (uint64, uint64) {
	var fixedInt uint64 = uint64(data[offset+7])<<56 |
		uint64(data[offset+6])<<48 |
		uint64(data[offset+5])<<40 |
		uint64(data[offset+4])<<32 |
		uint64(data[offset+3])<<24 |
		uint64(data[offset+2])<<16 |
		uint64(data[offset+1])<<8 |
		uint64(data[offset])
	return fixedInt, (offset + 8)
}

// Turns out we don't need to do length-encoded integers any more. Saving the code here
// in a comment in case it turns out to be relevant for future work.

// func encodedInt(data []byte, offset uint64) (uint64, uint64) {
// 	if data[offset] < 0xFB {
// 		value, newOffset := fixedInt1(data, offset)
// 		return uint64(value), newOffset
// 	} else if data[offset] == 0xFC {
// 		value, newOffset := fixedInt2(data, offset+1)
// 		return uint64(value), newOffset
// 	} else if data[offset] == 0xFD {
// 		value, newOffset := fixedInt3(data, offset+1)
// 		return uint64(value), newOffset
// 	} else if data[offset] == 0xFE {
// 		return fixedInt8(data, offset+1)
// 	} else {
// 		panic(fmt.Sprintf("Invalid header byte for length-encoded integer: 0x%x!", data[0]))
// 	}
// }

func (p *Packet) Dump() {
	output.Dump(p.Payload, "[%x] %d-byte %s packet (seq %d):\n", p.ConnectionId, len(p.Payload), p.packetTypeStr(), p.SequenceId)
}
