package main

import (
	"flag"
	"io"
	"log"
	"os"
)

const usageString = "Usage: mysql-query-logger [-h host] [-p port] [-i interface] [logfile]"

type Config struct {
	LogFile   *log.Logger // The logfile we're writing to
	MysqlHost string      // The host running MySQL
	MysqlPort int         // The MySQL server port on the MysqlHost
	Interface string      // The network interface to monitor
}

var defaultConfig Config = Config{
	nil,         // LogPath
	"localhost", // MysqlHost
	3306,        // MysqlPort
	"eth0",      // Interface
}

func GetConfig() Config {
	var conf Config

	// Open the filehandle for the logfile.
	switch len(flag.Args()) {
	case 0:
		conf.LogFile = makeLogger(os.Stdout)
	case 1:
		file, err := os.OpenFile(flag.Arg(0), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Can't open %s: %s", flag.Arg(0), err)
		}
		conf.LogFile = makeLogger(file)
	default:
		log.Fatal(usageString)
	}

	// Read the command-line flags.
	flag.StringVar(&conf.MysqlHost, "h", defaultConfig.MysqlHost, "The remote host running MySQL")
	flag.IntVar(&conf.MysqlPort, "p", defaultConfig.MysqlPort, "The MySQL server port on the remote host")
	flag.StringVar(&conf.Interface, "i", defaultConfig.Interface, "The network interface to monitor")
	flag.Parse()

	return conf
}

func makeLogger(handle io.Writer) *log.Logger {
	return log.New(handle, "", log.Ldate|log.Lmicroseconds|log.LUTC)
}
