package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := GetConfig()

	pcapHandle, err := pcap.OpenLive(conf.Interface, 1600, true, pcap.BlockForever)
	if err != nil {
		// The error contains the interface name already.
		log.Fatalf("Can't open network interface %s", err)
	}

	// TODO: Allow for monitoring multiple MySQL servers.
	filter := fmt.Sprintf("dst host %s and tcp port %d", conf.MysqlHost, conf.MysqlPort)
	err = pcapHandle.SetBPFFilter(filter)
	if err != nil {
		log.Fatalf("Can't apply pcap filter (%s): %s", filter, err)
	}

	packetSource := gopacket.NewPacketSource(pcapHandle, pcapHandle.LinkType())
	packetSource.DecodeOptions = gopacket.NoCopy
	packetChan := packetSource.Packets()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

Loop:
	for {
		select {
		case packet := <-packetChan:
			app := packet.ApplicationLayer()
			if app == nil {
				continue
			}
			payload := app.Payload()
			if payload == nil {
				continue
			}
			mysqlPacket := ReadMysqlPacket(payload)
			if mysqlPacket == nil {
				continue
			}
			conf.LogFile.Printf("%s;\n", mysqlPacket.Statement)

		case <-signalChan:
			break Loop
		}
	}

	printStatistics(pcapHandle)
}

func printStatistics(handle *pcap.Handle) {
	stats, err := handle.Stats()
	if err == nil {
		fmt.Fprintf(os.Stderr, "\nPackets received: %28d\nPackets dropped by kernel: %19d\nPackets dropped by interface: %16d\n",
			stats.PacketsReceived, stats.PacketsDropped, stats.PacketsIfDropped)
	} else {
		fmt.Fprintf(os.Stderr, "Couldn't read packet statistics: %s", err)
	}
}
