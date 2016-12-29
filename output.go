package main

import (
	"fmt"
	"log"
)

const (
	logLevelNormal  = 0
	logLevelVerbose = 1
	logLevelDebug   = 2
	logLevelDump    = 3
)

type Output struct {
	Logger *log.Logger
	Level  int
}

func NewOutput(config Config) (out Output) {
	out.Logger = log.New(config.LogFile, "", log.Ldate|log.Lmicroseconds|log.LUTC)
	out.Level = config.LogLevel
	return out
}

func (out Output) Dump(slice []byte, format string, args ...interface{}) {
	if out.Level >= logLevelDump {
		str := fmt.Sprintf(format, args...)
		rowCount := len(slice) / 16
		if len(slice)%16 > 0 {
			rowCount++
		}
		for i := 0; i < rowCount; i++ {
			str += "      "
			for j := 0; j < 16; j++ {
				if len(slice)-i*16 <= j {
					str += "   "
				} else {
					str += fmt.Sprintf("%2x ", slice[i*16+j])
				}
			}

			str += "      "
			for j := 0; j < 16; j++ {
				if len(slice)-i*16 <= j {
					str += "  "
				} else if slice[i*16+j] >= 0x20 && slice[i*16+j] <= 0x7e {
					str += fmt.Sprintf("%c ", slice[i*16+j])
				} else {
					str += "* "
				}
			}

			str += "\n"
		}
		out.Logger.Print(str)
	}
}

func (out Output) Debug(format string, args ...interface{}) {
	if out.Level >= logLevelDebug {
		out.Logger.Printf(format, args...)
	}
}

func (out Output) Verbose(format string, args ...interface{}) {
	if out.Level >= logLevelVerbose {
		out.Logger.Printf(format, args...)
	}
}

func (out Output) Log(format string, args ...interface{}) {
	out.Logger.Printf(format, args...)
}
