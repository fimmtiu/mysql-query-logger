package main

import (
	"github.com/google/gopacket"
	"time"
)

const (
	directionIncoming = iota
	directionOutgoing = iota
)

type Traffic struct {
	ConnectionId uint64
	Timestamp    time.Time
	Direction    uint8
	Data         []byte
}

func NewTraffic(netPacket gopacket.Packet, direction int) *Traffic {
	var t Traffic
	net := netPacket.NetworkLayer()
	if net == nil {
		return nil
	}
	app := netPacket.ApplicationLayer()
	if app == nil {
		return nil
	}

	t.ConnectionId = net.NetworkFlow().FastHash()
	t.Timestamp = netPacket.Metadata().CaptureInfo.Timestamp
	t.Direction = uint8(direction)
	t.Data = app.Payload()
	if t.Data == nil {
		return nil
	}

	return &t
}

func (t *Traffic) IsIncoming() bool {
	return t.Direction == directionIncoming
}

// Excises the first N bytes from this buffer and returns them.
func (t *Traffic) ShiftBytes(n uint64) []byte {
	slice := t.Data[:n]
	t.Data = t.Data[n:]
	output.Debug("Removed %d bytes from %s traffic", n, t.DirectionString())
	return slice
}

func (t *Traffic) DirectionString() string {
	if t.IsIncoming() {
		return "incoming"
	} else {
		return "outgoing"
	}
}

func (t *Traffic) Dump() {
	output.Dump(t.Data, "[%x] %d bytes of %s traffic:\n", t.ConnectionId, len(t.Data), t.DirectionString())
}
