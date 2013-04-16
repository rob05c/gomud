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
	"regexp"
	"crypto/rand"
	"code.google.com/p/go.crypto/ripemd160"
)

func handleCreatingPlayerPassVerify(managers metaManager, c net.Conn, player string, newPass string) {
	c.Write([]byte("Please verify your password.\n"))
	passVerify, err := getString(c)
	if err != nil {
		return
	}
	if newPass != passVerify {
		c.Write([]byte("The passwords you entered do not match.\n"))
		go handleCreatingPlayerPass(managers, c, player)
		return
	}
	fmt.Println("creating player")

 	salt := make([]byte, 128)
	n, err := rand.Read(salt)
	if n != len(salt) || err != nil {
		fmt.Println("Error creating salt.")
		c.Close()
		return
	}
	passBytes := []byte(newPass) // @todo remove this after fixing pass strings
	saltedPass := make([]byte, len(salt)+len(passBytes))
	saltedPass = append(saltedPass, salt...)
	saltedPass = append(saltedPass, passBytes...)
	h := ripemd160.New()
	h.Write(saltedPass)
	hashedPass := h.Sum(nil)

	newPlayer := player_state{name: player, pass: hashedPass, passthesalt: salt}
	managers.playerManager.createPlayer(newPlayer)
	playerRoomAdd<- struct{player string; roomId roomIdentifier} {player, 0}
	go handlePlayer(managers, c, player)
	return
}

func handleCreatingPlayerPass(managers metaManager, c net.Conn, player string) {
	c.Write([]byte("Please enter a password for your character.\n"))
	pass, err := getString(c)
	if err != nil {
		return
	}
	go handleCreatingPlayerPassVerify(managers, c, player, pass)
}
	
func handleCreatingPlayer(managers metaManager, c net.Conn, player string) {
	const playerCreateMessage = "No player by that name exists. Do you want to create a player?"
	c.Write([]byte(playerCreateMessage + "\n"))
	createReply, err := getString(c)
	if err != nil {
		return
	}
	if strings.HasPrefix(createReply, "y") {
		go handleCreatingPlayerPass(managers, c, player)
		return
	}
	go handleLogin(managers, c)
}

func handleLoginPass(managers metaManager, c net.Conn, playerName string) {
	c.Write([]byte("Please enter your password.\n"))
	pass, err := getBytesSecure(c)
	if err != nil {
		return
	}
	player, exists := managers.playerManager.getPlayer(playerName)
	if !exists {
		for i := range pass {
			pass[i] = 0
		}
		return // we just validated the player exists, so this shouldn't happen
	}
	
	saltedPass := make([]byte, len(player.passthesalt)+len(pass))
	saltedPass = append(saltedPass, player.passthesalt...)
	saltedPass = append(saltedPass, pass...)
	h := ripemd160.New()
	h.Write(saltedPass)
	hashedPass := h.Sum(nil)

	for i := range saltedPass {
		saltedPass[i] = 0
	}
	for i := range pass {
		pass[i] = 0
	}

	if !bytes.Equal(hashedPass, player.pass) {
		c.Write([]byte("Invalid password.\n"))
		c.Close()
		return
	}

	// player was found - user is now logged in
	const nameSuccessMessage = "You have been successfully logged in!"
	c.Write([]byte(nameSuccessMessage + "\n"))
	go handlePlayer(managers, c, playerName)
}

// this handles new connections, which are not yet logged in
func handleLogin(managers metaManager, c net.Conn) {
	negotiateTelnet(c)
	const loginMessageString = "Please enter your name:\n"
	c.Write([]byte(loginMessageString))
	for {
		player, error := getString(c)
		if error != nil {
			return
		}

		const validNameRegex = "[a-zA-Z]+" // names can only contain letters
		valid, err := regexp.MatchString(validNameRegex, player)
		if err != nil || !valid {
			const invalidNameMessage = "That is not a valid name. Please enter your name."
			c.Write([]byte(invalidNameMessage + "\n"))
			continue
		}

		// if the current player doesn't exist, assume the sent text is the player name
		_, playerExists := managers.playerManager .getPlayer(player)
		if !playerExists { // player wasn't found
			go handleCreatingPlayer(managers, c, player)
			return
		} 
		go handleLoginPass(managers, c, player)
		return
	}

}

// this handles connections for a logged-in player
func handlePlayer(managers metaManager, c net.Conn, player string) {
	c.Write([]byte("Welcome " + player + "!\n"))
	look(c, player, managers.roomManager)
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
			command(trimmedMessageArgs[1:], c, player, &managers)
		}
	}
}

/// when concatenating byte arrays, overwrites old bytes.
/// use for getting secure text, such as passwords.
func getBytesSecure(c net.Conn) ([]byte, error) {
	readBuf := make([]byte, 8)
	finalBuf := make([]byte, 0)
	for {
		n, err := c.Read(readBuf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("error: " + err.Error())
				return nil, err
			} else {
				fmt.Println("connection closed.")
				return nil, errors.New("connection closed")
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
	for i := range readBuf {
		readBuf[i] = 0
	}
	finalBuf = bytes.Trim(finalBuf, " \r\n")
	return finalBuf, nil
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
func listen(managers metaManager) {
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
		conn.Write([]byte("Welcome to gomud. "))
		go handleLogin(managers, conn)
	}
}
