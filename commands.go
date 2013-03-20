package main
import (
	"fmt"
	"net"
	"strconv"
	"strings"
)
const commandRejectMessage = "I don't understand."
var commands = map[string] func([]string, net.Conn, string)() {}

func walk(d Direction, c net.Conn, playerName string) {
	player, exists := getPlayer(playerName)
	if !exists {
		fmt.Println("walk called with nonplayer '" + playerName + "'")
		return
	}
	currentRoom, roomExists := getRoom(player.roomId)
	if !roomExists {
		fmt.Println("walk called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId))
		return
	}
	newRoomId, ok := currentRoom.exits[d]
	if !ok {
		const invalidDirectionMessage = "You faceplant a wall. Suck."
		c.Write([]byte(invalidDirectionMessage + "\n"))
		return
	}
	playerChange<- struct {key string; modify func(*player_state)} {player.name, func(player *player_state){
			player.roomId = newRoomId
			go look(c, playerName)
	}}
}

func look(c net.Conn, playerName string) {
	player, exists := getPlayer(playerName)
	if !exists {
		fmt.Println("look called with nonplayer '" + playerName + "'")
		return
	}

	currentRoom, roomExists := getRoom(player.roomId)
	if !roomExists {
		fmt.Println("look called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId))
		return
	}

	c.Write([]byte(currentRoom.Print() + "\n"))
}

func quicklook(c net.Conn, playerName string) {
	player, exists := getPlayer(playerName)
	if !exists {
		fmt.Println("quicklook called with nonplayer '" + playerName + "'")
		return
	}
	currentRoom, roomExists := getRoom(player.roomId)
	if !roomExists {
		fmt.Println("quicklook called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId)) 
		return
	}

	c.Write([]byte(currentRoom.PrintBrief() + "\n"))
}

func initCommandsAdmin(){
	commands["makeroom"] = func(args []string, c net.Conn, playerName string) {
		if len(args) < 2 {
			c.Write([]byte(commandRejectMessage + "3\n")) ///< @todo give better error
			return
		}
		newRoomDirection := stringToDirection(args[0])
		if newRoomDirection < north || newRoomDirection > southwest {
			c.Write([]byte(commandRejectMessage + "4\n")) ///< @todo give better error
			fmt.Println(args[0]) ///< @todo give more descriptive error
			fmt.Println(args[1]) ///< @todo give more descriptive error
			return
		}
		player, exists := getPlayer(playerName)
		if !exists {
			fmt.Println("makeroom called with nonplayer '" + playerName + "'")
			return
		}
		currentRoom, roomExists := getRoom(player.roomId)
		if !roomExists {
			fmt.Println("makeroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId)) 
			return
		}


		newRoomName := strings.Join(args[1:], " ")
		currentRoom.NewRoom(newRoomDirection, newRoomName, "")
		c.Write([]byte(newRoomName + " materializes to the " + newRoomDirection.String() + ". It is nondescript and seems as though it might fade away at any moment.\n"))
	}
	commands["mr"] = commands["makeroom"]

	commands["connectroom"] = func(args []string, c net.Conn, playerName string) {
		if len(args) < 2 {
			c.Write([]byte(commandRejectMessage + "5\n"))
			return
		}
		toConnectRoomId, err := strconv.Atoi(args[1])
		if err != nil {
			c.Write([]byte(commandRejectMessage + "6\n"))
			return
		}
		player, exists := getPlayer(playerName)
		if !exists {
			fmt.Println("connectroom called with nonplayer '" + playerName + "'")
			return
		}

		currentRoom, roomExists := getRoom(player.roomId)
		if !roomExists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId)) 
			return
		}

		newRoomDirection := stringToDirection(args[0])
		toConnectRoom, connectionRoomExists := getRoom(toConnectRoomId)
		if !connectionRoomExists {
			c.Write([]byte("No room exists with the given id.\n"))
			return
		}

		roomChange<- struct {id int; modify func(*room)} {currentRoom.id, func(r *room){
				r.exits[newRoomDirection] = toConnectRoom.id
		}}
		roomChange<- struct {id int; modify func(*room)} {toConnectRoom.id, func(r *room){
				r.exits[newRoomDirection.reverse()] = currentRoom.id
				go func() {
					c.Write([]byte("You become aware of a " + newRoomDirection.String() + " passage to " + toConnectRoom.name + ".\n"))
				}()
		}}
	}
	commands["cr"] = commands["connectroom"]

	commands["describeroom"] = func(args []string, c net.Conn, playerName string) {
		if len(args) < 1 {
			c.Write([]byte(commandRejectMessage + "3\n")) ///< @todo give better  error
			return
		}
		player, exists := getPlayer(playerName)
		if !exists {
			fmt.Println("describeroom called with nonplayer '" + playerName + "'")
			return
		}
		currentRoom, roomExists := getRoom(player.roomId)
		if !roomExists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId)) 
			return
		}
		roomChange<- struct {id int; modify func(*room)} {currentRoom.id, func(r *room){
				r.description = strings.Join(args[0:], " ")
				go func() {
					c.Write([]byte("Everything seems a bit more corporeal.\n"))
				}()
		}}
	}
	commands["dr"] = commands["describeroom"]
	// just displays the current room's ID. Probably doesn't need to be an admin command
	commands["roomid"] = func(args []string, c net.Conn, playerName string) {
		player, exists := getPlayer(playerName)
		if !exists {
			fmt.Println("describeroom called with nonplayer '" + playerName + "'")
			return
		}
		currentRoom, roomExists := getRoom(player.roomId)
		if !roomExists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(player.roomId)) 
			return
		}
		c.Write([]byte(strconv.Itoa(currentRoom.id) + "\n"))
	}
}

func initCommandsDirections() {
	commands["south"] = func(args []string, c net.Conn, player string) {
		walk(south, c, player)
	}
	commands["s"] = commands["south"]
	commands["north"] = func(args []string, c net.Conn, player string) {
		walk(north, c, player)
	}
	commands["n"] = commands["north"]
	commands["east"] = func(args []string, c net.Conn, player string) {
		walk(east, c, player)
	}
	commands["e"] = commands["east"]
	commands["west"] = func(args []string, c net.Conn, player string) {
		walk(west, c, player)
	}
	commands["w"] = commands["west"]
	commands["northeast"] = func(args []string, c net.Conn, player string) {
		walk(northeast, c, player)
	}
	commands["ne"] = commands["northeast"]
	commands["northwest"] = func(args []string, c net.Conn, player string) {
		walk(northwest, c, player)
	}
	commands["nw"] = commands["northwest"]
	commands["southeast"] = func(args []string, c net.Conn, player string) {
		walk(southeast, c, player)
	}
	commands["se"] = commands["southeast"]
	commands["southwest"] = func(args []string, c net.Conn, player string) {
		walk(southwest, c, player)
	}
	commands["sw"] = commands["southwest"]
}

func initCommands() {

	commands["look"] = func(args []string, c net.Conn, player string) {
		look(c, player)
	}
	commands["l"] = commands["look"]

	commands["quicklook"] = func(args []string, c net.Conn, player string) {
		quicklook(c, player)
	}
	commands["ql"] = commands["quicklook"]

	initCommandsDirections()
	initCommandsAdmin()
}
