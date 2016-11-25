package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"log"
)

func main() {
	conf := GetConfig()

	pcapHandle, err := pcap.OpenLive(conf.Interface, 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Can't open network interface %s: %s", conf.Interface, err)
	}

	// TODO: Allow for monitoring multiple MySQL servers.
	filter := fmt.Sprintf("dst host %s and tcp port %d", conf.MysqlHost, conf.MysqlPort)
	err = pcapHandle.SetBPFFilter(filter)
	if err != nil {
		log.Fatalf("Can't apply pcap filter (%s): %s", filter, err)
	}

	packetSource := gopacket.NewPacketSource(pcapHandle, pcapHandle.LinkType())
	packetSource.DecodeOptions = gopacket.NoCopy

	for packet := range packetSource.Packets() {
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
	}
}
