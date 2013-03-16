package main
import (
	"net"
)

var telnetState telnet_state

//
// telnet
//
type telnet_command byte
const (
	SE = 240
	NOP = 241
	DM = 242
	BRK = 243
	IP = 244
	AO = 245
	AYT = 246
	EC = 247
	EL = 248
	GA = 249
	SB = 250
	WILL = 251
	WONT = 252
	DO = 253
	DONT = 254
	IAC = 255
) 

type telnet_state struct {
	LocalEcho bool
}
 
func handleTelnet(option byte, optionInfo byte, c net.Conn) {
	// for now, reject all requests. We're very contrary.
	if option == DO || option == WILL {
		commandBytes := []byte{0, 0, 0}
		commandBytes[0] = IAC
		if option == DO {
			commandBytes[1] = WONT
		} else {
			commandBytes[1] = DONT
		}
		commandBytes[2] = optionInfo
		c.Write(commandBytes)
	}
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
