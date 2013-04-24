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
	managers.playerLocationManager.movePlayer(c, playerName, d, func(success bool) {
		if !success {
			// @todo tell the user why (no exit, blocked, etc.
			c.Write([]byte("You can't go there.\n"))
			return
		}
		go look(c, playerName, managers)
	})
}

func look(c net.Conn, playerName string, managers *metaManager) {
	roomId, exists := managers.playerLocationManager.playerRoom(playerName)
	if !exists {
		fmt.Println("look called with invalid  player'" + playerName + "'")
		return
	}
	currentRoom, exists := managers.roomManager.getRoom(roomId)
	if !exists {
		fmt.Println("look called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.Print(managers) + "\n"))
}

func quicklook(c net.Conn, playerName string, managers *metaManager) {
	roomId, exists := managers.playerLocationManager.playerRoom(playerName)
	if !exists {
		fmt.Println("quicklook called with invalid player  '" + playerName + "'")
		return
	}
	currentRoom, exists := managers.roomManager.getRoom(roomId)
	if !exists {
		fmt.Println("quicklook called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.PrintBrief(managers) + "\n"))
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

	// createitem name
	commands["createitem"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to create?\n"))
			return
		}
		itemName := args[0]
		it := item{
			id:     itemIdentifier(invalidIdentifier),
			name:   itemName,
			brief:  "An amorphous blob",
			long:   "What an incredibly ugly amorphous blob. The deities of creation really flubbed this one up.",
			ground: "A hideous amorphous blob quivers here."}
		id := managers.itemManager.createItem(it)
		player, exists := managers.playerManager.getPlayer(playerName)
		if !exists {
			fmt.Println("createitem called with invalid player '" + playerName + "'")
			return
		}
		managers.itemLocationManager.addItem(id, identifier(player.id), ilPlayer)
		c.Write([]byte("A " + itemName + " materialies in your hands.\n"))
	}

	commands["describeitem"] = func(args []string, c net.Conn, playerName string, managers *metaManager) {
		if len(args) < 2 {
			c.Write([]byte("What do you want to describe?\n"))
			return
		}

		itemInt, err := strconv.Atoi(args[0])
		if err != nil {
			c.Write([]byte("Please provide a valid identifier to describe.\n"))
			return
		}
		itemId := itemIdentifier(itemInt)
		it, exists := managers.itemManager.getItem(itemId)
		if !exists {
			c.Write([]byte("That does not exist.\n"))
			return
		}
		managers.itemManager.changeItem(itemId, func(i *item) {
			i.brief = strings.Join(args[1:], " ")
			c.Write([]byte("The " + it.name + " seems less ugly than it was.\n"))
		})
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

func initCommandsItems() {
	commands["get"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to get?\n"))
			return
		}

		realPlayer, exists := managers.playerManager.getPlayer(player)
		if !exists {
			fmt.Println("get called with invalid player2 '" + player + "'")
			return
		}

		roomId, exists := managers.playerLocationManager.playerRoom(player)
		if !exists {
			fmt.Println("get called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("get called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		itemInt, err := strconv.Atoi(args[0])
		if err != nil {
			// getting by name, not id
			items := managers.itemLocationManager.locationItems(identifier(roomId), ilRoom)
			for _, itemId := range items {
				it, exists := managers.itemManager.getItem(itemId)
				if !exists {
					fmt.Println("get got nonexistent item from itemLocationManager '" + itemId.String() + "'")
				}
				if it.name == args[0] {
					managers.itemLocationManager.moveItem(c, it.id, identifier(roomId), ilRoom, realPlayer.id, ilPlayer, func(success bool) {
						if success {
							c.Write([]byte("You pick up " + it.brief + ".\n"))
						} else {
							c.Write([]byte("That is not here.\n"))
						}
					})
					return
				}
			}
			c.Write([]byte("That is not here.\n"))
			return
		}
		fmt.Println("debug got " + strconv.Itoa(itemInt))
		it, exists := managers.itemManager.getItem(itemIdentifier(itemInt))
		if !exists {
			c.Write([]byte("That does not exist.\n"))
			return
		}

		managers.itemLocationManager.moveItem(c, it.id, identifier(currentRoom.id), ilRoom, realPlayer.id, ilPlayer, func(success bool) {
			if success {
				c.Write([]byte("You pick up " + it.brief + ".\n"))
			} else {
				c.Write([]byte("That is not here.\n"))
			}
		})
	}
	commands["drop"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to drop?\n"))
			return
		}

		roomId, exists := managers.playerLocationManager.playerRoom(player)
		if !exists {
			fmt.Println("drop called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("drop called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		realPlayer, exists := managers.playerManager.getPlayer(player)
		if !exists {
			fmt.Println("Drop called with invalid player2 '" + player + "'")
			return
		}

		itemInt, err := strconv.Atoi(args[0])
		if err != nil {
			// getting by name, not id
			items := managers.itemLocationManager.locationItems(identifier(realPlayer.id), ilPlayer)
			for _, itemId := range items {
				it, exists := managers.itemManager.getItem(itemId)
				if !exists {
					fmt.Println("drop got nonexistent item from itemLocationManager '" + itemId.String() + "'")
				}
				if it.name == args[0] {
					managers.itemLocationManager.moveItem(c, it.id, realPlayer.id, ilPlayer, identifier(currentRoom.id), ilRoom, func(success bool) {
						if success {
							c.Write([]byte("You drop " + it.brief + ".\n"))
						} else {
							c.Write([]byte("You are not holding that.\n"))
						}
					})
					return
				}
			}
			c.Write([]byte("You are not holding that.\n"))
			return
		}

		it, exists := managers.itemManager.getItem(itemIdentifier(itemInt))
		if !exists {
			c.Write([]byte("That does not exist.\n"))
			return
		}

		managers.itemLocationManager.moveItem(c, it.id, realPlayer.id, ilPlayer, identifier(currentRoom.id), ilRoom, func(success bool) {
			if success {
				c.Write([]byte("You drop " + it.brief + ".\n"))
			} else {
				c.Write([]byte("You aren't holding that.\n"))
			}
		})
	}

	commands["items"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		realPlayer, exists := managers.playerManager.getPlayer(player)
		if !exists {
			fmt.Println("Drop called with invalid player2 '" + player + "'")
			return
		}

		items := managers.itemLocationManager.locationItems(realPlayer.id, ilPlayer)
		for _, itemId := range items {
			it, exists := managers.itemManager.getItem(itemId)
			if !exists {
				fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			c.Write([]byte(it.id.String() + it.name + "\n"))
		}
	}
	commands["ii"] = commands["items"]

	commands["inventory"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		realPlayer, exists := managers.playerManager.getPlayer(player)
		if !exists {
			fmt.Println("inventory called with invalid player2 '" + player + "'")
			return
		}
		items := managers.itemLocationManager.locationItems(realPlayer.id, ilPlayer)
		if len(items) == 0 {
			c.Write([]byte("You aren't carrying anything.\n"))
			return
		}
		s := "You are carrying "
		if len(items) == 1 {
			it, exists := managers.itemManager.getItem(items[0])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[0].String() + "'")
				s += "nothing of interest."
				c.Write([]byte(s))
				return
			}
			s += it.brief
			s += ".\n"
			c.Write([]byte(s))
			return
		}
		if len(items) == 2 {
			it, exists := managers.itemManager.getItem(items[0])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[0].String() + "'")
				s += "nothing of interest"
			} else {
				s += it.brief
			}
			s += ", "
			it, exists = managers.itemManager.getItem(items[1])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[1].String() + "'")
				s += "nothing of interest"
			} else {
				s += it.brief
			}
			s += ".\n"
			c.Write([]byte(s))
			return
		}

		lastItem := items[len(items)-1]
		items = items[:len(items)-1]
		for _, itemId := range items {
			it, exists := managers.itemManager.getItem(itemId)
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			s += it.brief + ", "
		}
		it, exists := managers.itemManager.getItem(lastItem)
		if !exists {
			fmt.Println("inventory got nonexistent item from itemLocationManager '" + lastItem.String() + "'")
			s += "nothing of interest"
		} else {
			s += it.brief
		}
		s += ".\n"
		c.Write([]byte(s))
	}
	commands["inv"] = commands["inventory"]
	commands["i"] = commands["inventory"]

	commands["itemshere"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		roomId, exists := managers.playerLocationManager.playerRoom(player)
		if !exists {
			fmt.Println("items called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := managers.roomManager.getRoom(roomId)
		if !exists {
			fmt.Println("items called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		items := managers.itemLocationManager.locationItems(identifier(currentRoom.id), ilRoom)
		for _, itemId := range items {
			it, exists := managers.itemManager.getItem(itemId)
			if !exists {
				fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			c.Write([]byte(it.id.String() + it.name + "\t" + it.brief + "\n"))
		}
	}
	commands["ih"] = commands["itemshere"]

}

func initCommands() {
	commands["look"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		look(c, player, managers)
	}
	commands["l"] = commands["look"]

	commands["quicklook"] = func(args []string, c net.Conn, player string, managers *metaManager) {
		quicklook(c, player, managers)
	}
	commands["ql"] = commands["quicklook"]

	initCommandsDirections()
	initCommandsItems()
	initCommandsAdmin()
}
