package main

import (
	"fmt"
	"strings"
	"time"
)

// TODO: We'll need to experiment with these numbers -- how often will the channels block?
const conversationChannelSize = 100
const dataChannelSize = 100

const cleanupInterval = 30 * time.Second
const conversationTimeout = 3 * time.Minute
const trafficTimeout = 3 * time.Minute

const (
	statusIncomplete = iota // The entire query hasn't arrived yet (>16MB)
	statusWaiting    = iota // The query was sent and we're waiting for a response
	statusDone       = iota // The query was successfully processed
	statusError      = iota // The query failed
)

type TrafficSet struct {
	streams [2]*Traffic // separate incoming and outgoing streams for each connection
}

type Conversation struct {
	Statement   string
	Status      uint8
	ElapsedTime time.Duration
	executedAt  time.Time
	createdAt   time.Time
}

type PacketReader struct {
	Channel       chan *Conversation
	listener      *NetListener
	config        Config
	conversations map[uint64]*Conversation
	traffic       map[uint64]TrafficSet
}

func NewPacketReader(config Config, listener *NetListener) *PacketReader {
	conversationChan := make(chan *Conversation, conversationChannelSize)
	return &PacketReader{conversationChan, listener, config, make(map[uint64]*Conversation), make(map[uint64]TrafficSet)}
}

func (pr *PacketReader) Read() {
	cleanupTimer := time.Tick(cleanupInterval)
	trafficChannel := make(chan *Traffic, dataChannelSize)
	go pr.listener.Listen(trafficChannel)

	for {
		select {
		case newTraffic := <-trafficChannel:
			traffic := pr.storeTraffic(newTraffic)

			for {
				// Read new packets until we run out of input
				packet := NewPacket(traffic)
				if packet == nil {
					break
				}
				packet.Dump()
				if !pr.isPacketRelevant(packet) {
					output.Debug("Not relevant; skipping.")
					continue
				}

				c := pr.getConversation(traffic.ConnectionId)
				if packet.ContainsQuery() {
					c.Statement = strings.Join([]string{c.Statement, packet.Statement()}, "")
					if !packet.Partial {
						c.Status = statusWaiting
					}
				}
				if !packet.Partial && packet.ExecutesQuery() {
					c.executedAt = time.Now()
				}

				if packet.IsResponse() {
					if c.Status != statusWaiting {
						panic(fmt.Sprintf("Weird status for conversation: %d", c.Status))
					}
					c.ElapsedTime = time.Since(c.executedAt)
					if packet.IsErrorResponse() {
						c.Status = statusError
					} else {
						c.Status = statusDone
					}

					delete(pr.conversations, packet.ConnectionId)
					output.Debug("Deleted conversation for connection %x", packet.ConnectionId)
					pr.Channel <- c
				}
			}

		case <-cleanupTimer:
			t := time.Now()
			pr.clean()
			output.Verbose("Cleanup took %d ms. (%d traffic streams, %d conversations)", time.Since(t).Nanoseconds()/1000000, len(pr.traffic), len(pr.conversations))
		}
	}
}

// Fetch the requested conversation, or create one if it doesn't exist yet.
func (pr *PacketReader) getConversation(connId uint64) *Conversation {
	_, exists := pr.conversations[connId]
	if !exists {
		pr.conversations[connId] = &Conversation{"", statusIncomplete, 0, time.Time{}, time.Now()}
		output.Debug("Created conversation %p for connection %x", pr.conversations[connId], connId)
	}
	return pr.conversations[connId]
}

// Append some on-the-wire data for a connection to a stream.
func (pr *PacketReader) storeTraffic(t *Traffic) *Traffic {
	set, exists := pr.traffic[t.ConnectionId]
	if exists && set.streams[t.Direction] != nil {
		set.streams[t.Direction].Timestamp = t.Timestamp
		output.Debug("existing %s data before: %d bytes", t.DirectionString(), len(set.streams[t.Direction].Data))
		set.streams[t.Direction].Data = append(set.streams[t.Direction].Data, t.Data...)
		output.Debug("existing %s data after: %d bytes", t.DirectionString(), len(set.streams[t.Direction].Data))
	} else {
		output.Debug("new %s data after: %d bytes", t.DirectionString(), len(t.Data))
		set.streams[t.Direction] = t
	}
	pr.traffic[t.ConnectionId] = set
	return set.streams[t.Direction]
}

// We want to ignore any outgoing packets for connections that don't have a
// Conversation, because that's just query results that we're not going to
// log. Outgoing packets on completed conversations should be a "can't
// happen", because the conversation should have already been removed.
func (pr *PacketReader) isPacketRelevant(p *Packet) bool {
	if p.IsResponse() {
		c, exists := pr.conversations[p.ConnectionId]
		if exists && c.Status != statusWaiting {
			panic(fmt.Sprintf("Outgoing traffic on a completed Conversation (%p, connection %x)! (status %d)", c, p.ConnectionId, c.Status))
		}
		return exists
	}
	return p.IsRelevantCommand()
}

// Periodically clean out any stale traffic and conversations. I'm aware that
// this cleanup work effectively constitutes a "stop the world" garbage
// collector, and if it causes more than a trivial amount of latency in
// real-world conditions, we'll need to do something more complicated.
func (pr *PacketReader) clean() {
	start := time.Now()

	trafficThreshold := start.Add(-trafficTimeout)
	for connId, set := range pr.traffic {
		for dir := 0; dir < len(set.streams); dir++ {
			traffic := set.streams[dir]
			if traffic != nil && traffic.Timestamp.Before(trafficThreshold) {
				output.Debug("Deleting old %s traffic for connection %x (%d bytes)", traffic.DirectionString(), connId, len(traffic.Data))
				set.streams[dir] = nil
				if set.streams[0] == nil && set.streams[1] == nil {
					delete(pr.traffic, connId)
				} else {
					pr.traffic[connId] = set
				}
			}
		}
	}

	conversationThreshold := start.Add(-conversationTimeout)
	for connId, conversation := range pr.conversations {
		if conversation.createdAt.Before(conversationThreshold) {
			output.Debug("Deleting old conversation for connection %x", connId)
			delete(pr.conversations, connId)
		}
	}
}
