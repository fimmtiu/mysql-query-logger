package main

import (
	"os"
	"os/signal"
	"syscall"
)

var output Output

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	config := GetConfig()
	output = NewOutput(config)
	listener := NewNetListener(config)
	reader := NewPacketReader(config, listener)
	go reader.Read()

Loop:
	for {
		select {
		case conversation := <-reader.Channel:
			output.Log("%6fs: %s", conversation.ElapsedTime.Seconds(), conversation.Statement)

		case <-signalChan:
			break Loop
		}
	}

	listener.PrintStatistics(os.Stderr)
}
