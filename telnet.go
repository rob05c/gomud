package main

import (
	"net"
)

//
// telnet
//
type telnet_command byte

const (
	SE   = 240
	NOP  = 241
	DM   = 242
	BRK  = 243
	IP   = 244
	AO   = 245
	AYT  = 246
	EC   = 247
	EL   = 248
	GA   = 249
	SB   = 250
	WILL = 251
	WONT = 252
	DO   = 253
	DONT = 254
	IAC  = 255
)

type telnet_state struct {
	LocalEcho bool
}

func handleTelnet(recieved []byte, c net.Conn) []byte {
	if len(recieved) == 0 {
		return recieved
	}
	if recieved[0] != IAC {
		return recieved
	}
	recieved = recieved[1:]
	var option byte
	if len(recieved) > 1 {
		option = recieved[0]
		recieved = recieved[1:]
	} else {
		option = byte(0)
	}
	// for now, reject all requests. We're very contrary.
	if option == DO || option == WILL {
		if option == DO {
			c.Write([]byte{IAC, WONT, option})
		} else {
			c.Write([]byte{IAC, DONT, option})
		}
	}
	return recieved
	/*
		var optionInfo byte
		if len(readBuf) > 2 {
			optionInfo = readBuf[2]
		} else {
			optionInfo = byte(0)
		}
	*/
}

func telnetCommandBytes(command telnet_command, option byte) []byte {
	commandBytes := make([]byte, 0) // 3 or 4?
	commandBytes = append(commandBytes, IAC)
	commandBytes = append(commandBytes, byte(command))
	commandBytes = append(commandBytes, option)
	return commandBytes
}

func negotiateTelnet(c net.Conn) {
	//	fmt.Println("sending" + string(telnetCommandBytes(WILL, 1)))
	//	c.Write(telnetCommandBytes(DO, 1)) // IAC WILL ECHO
}
