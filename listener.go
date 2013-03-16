package main
import (
	"fmt"
	"net"
	"strconv"
	"io"
	"bytes"
	"os"
	"strings"
	"errors"
)

// this handles new connections, which are not yet logged in
func handleLogin(c net.Conn) {
	negotiateTelnet(c)
	const loginMessageString = "Welcome to the gomud. Please enter your name:\n"
	c.Write([]byte(loginMessageString))
	for {
		player, error := getString(c)
		if error != nil {
			return
		}
		// if the current player doesn't exist, assume the sent text is the player name
		_, playerExists := getPlayer(player)
		if !playerExists { // player wasn't found
			const nameQueryMessage = "What is your name?"
			const nameQueryRejectMessage = "No player by that name exists. " + nameQueryMessage
			c.Write([]byte(nameQueryRejectMessage + "\n"))
			continue
		} 
		// player was found - user is now logged in
		const nameSuccessMessage = "You have been successfully logged in!"
		c.Write([]byte(nameSuccessMessage + "\n"))
		look(c, player)
		go handlePlayer(c, player)
		return
	}

}

// this handles connections for a logged-in player
func handlePlayer(c net.Conn, player string) {
	for {
		message, error := getString(c)
		if error != nil {
			return
		}

		messageArgs := strings.Split(message, " ")
		var trimmedMessageArgs []string // accomodates extra spaces between args
		for _, arg := range messageArgs {
			trimmedArg := strings.Trim(arg, " ")
			if trimmedArg != "" {
				trimmedMessageArgs = append(trimmedMessageArgs, trimmedArg)
			}
		}

		if len(trimmedMessageArgs) == 0 {
			c.Write([]byte(commandRejectMessage + "1\n"))
			continue
		}
		if command, commandExists := commands[trimmedMessageArgs[0]]; !commandExists {
			c.Write([]byte(commandRejectMessage + "2\n"))
		} else {
			command(trimmedMessageArgs[1:], c, player)
		}
	}
}

// this returns a string sent by the connected client.
// it also processes any Telnet commands it happens to read
func getString(c net.Conn) (string, error) {
	//debug
	fi, _ := os.Create("rec")
	defer fi.Close()

	readBuf := make([]byte, 8)
	finalBuf := make([]byte, 0)
	for {
		n, err := c.Read(readBuf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("error: " + err.Error())
				return "", err
			} else {
				fmt.Println("connection closed.")
				return "", errors.New("connection closed")
			}
		}
		readBuf = bytes.Trim(readBuf[:n], "\x00")
		if len(readBuf) == 0 {
			continue
		}
		if readBuf[0] == IAC {
			var option byte
			if len(readBuf) > 1 {
				option = readBuf[0]
			} else {
				option = byte(0)
			}
			
			var optionInfo byte
			if len(readBuf) > 2 {
				optionInfo = readBuf[2]
			} else {
				optionInfo = byte(0)
			}
			handleTelnet(option, optionInfo, c)
			continue
		}
		finalBuf = append(finalBuf, readBuf...)
		if len(finalBuf) == 0 {
			continue
		}
		if finalBuf[len(finalBuf)-1] == '\n' {
			break;
		}
	}
	finalBuf = bytes.Trim(finalBuf, " \r\n")
	fi.WriteString(string(finalBuf))
	fmt.Println("read " + strconv.Itoa(len(finalBuf)) + " bytes: B" + string(finalBuf) + "B")
	return string(finalBuf), nil
}

// listen for new connections, and spin them off into goroutines
func listen() {
	port := defaultPort
	if len(os.Args) > 1 {
		argPort, success := strconv.Atoi(os.Args[1])
		if success != nil {
			port = defaultPort
		} else if port < 0 || port > 65535 {
			port = defaultPort
		} else {
			port = argPort
		}
	}
	ln, err := net.Listen("tcp", ":" + strconv.Itoa(port))
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}
	fmt.Println("running at " + ln.Addr().String())
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleLogin(conn)
	}
}
