package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const commandRejectMessage = "I don't understand."

var commands = map[string]func([]string, net.Conn, string, *metaManager){}

func walk(d Direction, c net.Conn, playerName string, managers *metaManager) {
	managers.playerLocationManager.movePlayer(c, playerName, d, managers.roomManager)
}

func look(c net.Conn, playerName string, playerLocations *playerLocationManager, rooms *roomManager) {
	roomId, exists := playerLocations.playerRoom(playerName)
	if !exists {
		fmt.Println("look called with invalid  player'" + playerName + "'")
		return
	}
	currentRoom, exists := rooms.getRoom(roomId)
	if !exists {
		fmt.Println("look called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.Print() + "\n"))
}

func quicklook(c net.Conn, playerName string, playerLocations *playerLocationManager, rooms *roomManager) {
	roomId, exists := playerLocations.playerRoom(playerName)
	if !exists {
		fmt.Println("quicklook called with invalid player  '" + playerName + "'")
		return
	}
	currentRoom, exists := rooms.getRoom(roomId)
	if !exists {
		fmt.Println("quicklook called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.PrintBrief() + "\n"))
}

func initCommandsAdmin() {
	commands["makeroom"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		if len(args) < 2 {
			c.Write([]byte(commandRejectMessage + "3\n")) ///< @todo give better error
			return
		}
		newRoomDirection := stringToDirection(args[0])
		if newRoomDirection < north || newRoomDirection > southwest {
			c.Write([]byte(commandRejectMessage + "4\n")) ///< @todo give better error
			fmt.Println(args[0])                          ///< @todo give more descriptive error
			fmt.Println(args[1])                          ///< @todo give more descriptive error
			return
		}
		roomId, exists := managers.playerLocationManager.playerRoom(playerName)
		if !exists {
			fmt.Println("makeroom called with invalid player '" + playerName + "'")
			return
		}
		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("makeroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}

		newRoomName := strings.Join(args[1:], " ")
		currentRoom.NewRoom(managers.roomManager, newRoomDirection, newRoomName, "")
		c.Write([]byte(newRoomName + " materializes to the " + newRoomDirection.String() + ". It is nondescript and seems as though it might fade away at any moment.\n"))
	}
	commands["mr"] = commands["makeroom"]

	commands["connectroom"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		if len(args) < 2 {
			c.Write([]byte(commandRejectMessage + "5\n"))
			return
		}
		toConnectRoomIdInt, err := strconv.Atoi(args[1])
		if err != nil {
			c.Write([]byte(commandRejectMessage + "6\n"))
			return
		}
		toConnectRoomId := roomIdentifier(toConnectRoomIdInt)
		roomId, exists := managers.playerLocationManager.playerRoom(playerName)
		if !exists {
			fmt.Println("connectroom called with invalid player '" + playerName + "'")
			return
		}

		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}

		newRoomDirection := stringToDirection(args[0])
		toConnectRoom, connectionRoomExists := managers.roomManager.getRoom(toConnectRoomId)
		if !connectionRoomExists {
			c.Write([]byte("No room exists with the given id.\n"))
			return
		}

		managers.roomManager.changeRoom(currentRoom.id, func(r *room) {
			r.exits[newRoomDirection] = toConnectRoom.id
		})
		managers.roomManager.changeRoom(toConnectRoom.id, func(r *room) {
			r.exits[newRoomDirection.reverse()] = currentRoom.id
			go func() {
				c.Write([]byte("You become aware of a " + newRoomDirection.String() + " passage to " + toConnectRoom.name + ".\n"))
			}()
		})
	}
	commands["cr"] = commands["connectroom"]

	commands["describeroom"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		if len(args) < 1 {
			c.Write([]byte(commandRejectMessage + "3\n")) ///< @todo give better  error
			return
		}
		roomId, exists := managers.playerLocationManager.playerRoom(playerName)
		if !exists {
			fmt.Println("describeroom called with invalid player '" + playerName + "'")
			return
		}
		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}
		managers.roomManager.changeRoom(currentRoom.id, func(r *room) {
			r.description = strings.Join(args[0:], " ")
			go func() {
				c.Write([]byte("Everything seems a bit more corporeal.\n"))
			}()
		})
	}
	commands["dr"] = commands["describeroom"]
	// just displays the current room's ID. Probably doesn't need to be an admin command
	commands["roomid"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		roomId, exists := managers.playerLocationManager.playerRoom(playerName)
		if !exists {
			fmt.Println("roomid called with invalid player '" + playerName + "'")
			return
		}

		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}
		c.Write([]byte(strconv.Itoa(int(currentRoom.id)) + "\n"))
	}
}

func initCommandsDirections() {
	commands["south"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(south, c, player, managers)
	}
	commands["s"] = commands["south"]
	commands["north"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(north, c, player, managers)
	}
	commands["n"] = commands["north"]
	commands["east"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(east, c, player, managers)
	}
	commands["e"] = commands["east"]
	commands["west"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(west, c, player, managers)
	}
	commands["w"] = commands["west"]
	commands["northeast"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(northeast, c, player, managers)
	}
	commands["ne"] = commands["northeast"]
	commands["northwest"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(northwest, c, player, managers)
	}
	commands["nw"] = commands["northwest"]
	commands["southeast"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(southeast, c, player, managers)
	}
	commands["se"] = commands["southeast"]
	commands["southwest"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		walk(southwest, c, player, managers)
	}
	commands["sw"] = commands["southwest"]
}

func initCommands() {
	commands["look"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		look(c, player, managers.playerLocationManager, managers.roomManager)
	}
	commands["l"] = commands["look"]

	commands["quicklook"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		quicklook(c, player, managers.playerLocationManager, managers.roomManager)
	}
	commands["ql"] = commands["quicklook"]

	initCommandsDirections()
	initCommandsAdmin()
}
