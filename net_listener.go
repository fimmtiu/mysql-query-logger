package main

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"io"
	"log"
	"net"
)

// Read all the things!
const Mtu = 32000

type NetListener struct {
	handle     *pcap.Handle
	source     *gopacket.PacketSource
	serverHost gopacket.Endpoint
	serverPort gopacket.Endpoint
}

func NewNetListener(config Config) *NetListener {
	var listener NetListener
	var err error

	listener.handle, err = pcap.OpenLive(config.Interface, Mtu, true, pcap.BlockForever)
	if err != nil {
		// The error contains the interface name already.
		log.Fatalf("Can't open network interface %s", err)
	}

	filter := fmt.Sprintf("host %s and tcp port %d", config.MysqlHost, config.MysqlPort)
	err = listener.handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatalf("Can't apply pcap filter (%s): %s", filter, err)
	}

	listener.source = gopacket.NewPacketSource(listener.handle, listener.handle.LinkType())
	listener.source.DecodeOptions = gopacket.NoCopy
	listener.serverHost, err = getEndpointHost(config.MysqlHost)
	if err != nil {
		log.Fatalf("Can't look up MySQL host %s: %s", config.MysqlHost, err)
	}
	listener.serverPort = getEndpointPort(config.MysqlPort)

	return &listener
}

// Infinitely loop over the stream of pcap packets and convert them to Traffic objects.
func (listener *NetListener) Listen(receiver chan *Traffic) {
	output.Verbose("Started listening...")

	// TODO: Is this faster with gopacket.DecodingLayerParser?

	for netPacket := range listener.source.Packets() {
		direction := listener.getDirection(netPacket)
		if direction >= 0 {
			traffic := NewTraffic(netPacket, direction)
			if traffic != nil {
				// traffic.Dump()
				receiver <- traffic
			}
		}
	}
}

// Prints some pcap statistics over the daemon's runtime.
func (listener *NetListener) PrintStatistics(output io.Writer) {
	stats, err := listener.handle.Stats()
	if err == nil {
		fmt.Fprintf(output, "\nPackets received: %28d\nPackets dropped by kernel: %19d\nPackets dropped by interface: %16d\n",
			stats.PacketsReceived, stats.PacketsDropped, stats.PacketsIfDropped)
	} else {
		fmt.Fprintf(output, "Couldn't read packet statistics: %s", err)
	}
}

// Converts the host to a network-layer endpoint.
func getEndpointHost(host string) (endpt gopacket.Endpoint, err error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return endpt, err
	}
	ip := ips[0].To4()
	if ip == nil {
		return endpt, errors.New("FIXME: No IPv6 support yet!")
	}
	return layers.NewIPEndpoint(ip), nil
}

// Converts the port to a transport-layer endpoint.
func getEndpointPort(port int) (endpt gopacket.Endpoint) {
	return layers.NewTCPPortEndpoint(layers.TCPPort(port))
}

// Looks at the destination endpoint and returns the constant representing
// the direction of the traffic.
func (listener *NetListener) getDirection(netPacket gopacket.Packet) int {
	network := netPacket.NetworkLayer()
	transport := netPacket.TransportLayer()
	if network == nil || transport == nil {
		return -1
	}
	if network.NetworkFlow().Dst() == listener.serverHost && transport.TransportFlow().Dst() == listener.serverPort {
		return directionIncoming
	} else {
		return directionOutgoing
	}
}
