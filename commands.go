package main

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const commandRejectMessage = "I don't understand."

var commands = map[string]func([]string, net.Conn, string, *metaManager){}

func say(message string, player string, world *metaManager) {
	if len(message) == 0 {
		return
	}
	rId, exists := world.playerLocations.playerRoom(player)
	if !exists {
		fmt.Println("say error: playerRoom got nonexistent room for " + player)
		return
	}
	r, exists := world.rooms.getRoom(rId)
	if !exists {
		fmt.Println("say error: getRoom got nonexistent room " + rId.String())
		return
	}

	p, exists := world.players.getPlayer(player)
	if !exists {
		fmt.Println("say error: getPlayer got nonexistent player " + player)
		return
	}

	sentenceEnd, err := regexp.Compile(`[.!?]$`) // @todo make this locale aware.
	if err != nil {
		fmt.Println(err)
		return
	}
	if !sentenceEnd.Match([]byte(message)) {
		message += "."
	}
	message = strings.ToUpper(string(message[0])) + message[1:]
	roomMessage := Pink + player + " says, \"" + message + "\"" + Reset // @todo make this locale aware, << >> vs " " vs ' '
	selfMessage := Pink + "You say, \"" + message + "\"" + Reset
	go r.Write(roomMessage, world.playerLocations, player)
	go p.Write(selfMessage)
}

func walk(d Direction, c net.Conn, playerName string, world *metaManager) {
	world.playerLocations.movePlayer(c, playerName, d, func(success bool) {
		if !success {
			// @todo tell the user why (no exit, blocked, etc.
			c.Write([]byte("You can't go there.\n"))
			return
		}
		go look(c, playerName, world)
	})
}

func look(c net.Conn, playerName string, world *metaManager) {
	roomId, exists := world.playerLocations.playerRoom(playerName)
	if !exists {
		fmt.Println("look called with invalid  player'" + playerName + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("look called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.Print(world, playerName) + "\n"))
}

func quicklook(c net.Conn, playerName string, world *metaManager) {
	roomId, exists := world.playerLocations.playerRoom(playerName)
	if !exists {
		fmt.Println("quicklook called with invalid player  '" + playerName + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("quicklook called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
		return
	}
	c.Write([]byte(currentRoom.PrintBrief(world, playerName) + "\n"))
}

func initCommandsAdmin() {
	commands["makeroom"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
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
		roomId, exists := world.playerLocations.playerRoom(playerName)
		if !exists {
			fmt.Println("makeroom called with invalid player '" + playerName + "'")
			return
		}
		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("makeroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}

		newRoomName := strings.Join(args[1:], " ")
		currentRoom.NewRoom(world.rooms, newRoomDirection, newRoomName, "")
		c.Write([]byte(newRoomName + " materializes to the " + newRoomDirection.String() + ". It is nondescript and seems as though it might fade away at any moment.\n"))
	}
	commands["mr"] = commands["makeroom"]

	commands["connectroom"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
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
		roomId, exists := world.playerLocations.playerRoom(playerName)
		if !exists {
			fmt.Println("connectroom called with invalid player '" + playerName + "'")
			return
		}

		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}

		newRoomDirection := stringToDirection(args[0])
		toConnectRoom, connectionRoomExists := world.rooms.getRoom(toConnectRoomId)
		if !connectionRoomExists {
			c.Write([]byte("No room exists with the given id.\n"))
			return
		}

		world.rooms.changeRoom(currentRoom.id, func(r *room) {
			r.exits[newRoomDirection] = toConnectRoom.id
		})
		world.rooms.changeRoom(toConnectRoom.id, func(r *room) {
			r.exits[newRoomDirection.reverse()] = currentRoom.id
			go func() {
				c.Write([]byte("You become aware of a " + newRoomDirection.String() + " passage to " + toConnectRoom.name + ".\n"))
			}()
		})
	}
	commands["cr"] = commands["connectroom"]

	commands["describeroom"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
		if len(args) < 1 {
			c.Write([]byte(commandRejectMessage + "3\n")) ///< @todo give better  error
			return
		}
		roomId, exists := world.playerLocations.playerRoom(playerName)
		if !exists {
			fmt.Println("describeroom called with invalid player '" + playerName + "'")
			return
		}
		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}
		world.rooms.changeRoom(currentRoom.id, func(r *room) {
			r.description = strings.Join(args[0:], " ")
			go func() {
				c.Write([]byte("Everything seems a bit more corporeal.\n"))
			}()
		})
	}
	commands["dr"] = commands["describeroom"]
	// just displays the current room's ID. Probably doesn't need to be an admin command
	commands["roomid"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
		roomId, exists := world.playerLocations.playerRoom(playerName)
		if !exists {
			fmt.Println("roomid called with invalid player '" + playerName + "'")
			return
		}

		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("connectroom called with player with invalid room '" + playerName + "' " + strconv.Itoa(int(roomId)))
			return
		}
		c.Write([]byte(strconv.Itoa(int(currentRoom.id)) + "\n"))
	}

	// createitem name
	commands["createitem"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to create?\n"))
			return
		}
		itemName := args[0]
		var it item = genericItem{
			id:    itemIdentifier(invalidIdentifier),
			name:  itemName,
			brief: "An amorphous blob"}

		id := world.items.createItem(it)
		player, exists := world.players.getPlayer(playerName)
		if !exists {
			fmt.Println("createitem called with invalid player '" + playerName + "'")
			return
		}
		world.itemLocations.addItem(id, identifier(player.id), ilPlayer)
		c.Write([]byte("A " + itemName + " materialises in your hands.\n"))
	}
	commands["ci"] = commands["createitem"]

	commands["createnpc"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to create?\n"))
			return
		}
		itemName := args[0]
		var it item = npc{
			id:    itemIdentifier(invalidIdentifier),
			name:  itemName,
			brief: "A mysterious figure"}

		id := world.items.createItem(it)
		player, exists := world.players.getPlayer(playerName)
		if !exists {
			fmt.Println("createnpc called with invalid player '" + playerName + "'")
			return
		}
		world.itemLocations.addItem(id, identifier(player.id), ilPlayer)
		c.Write([]byte("A " + itemName + " materialises in your hands.\n"))
	}
	commands["cn"] = commands["createnpc"]

	commands["describeitem"] = func(args []string, c net.Conn, playerName string, world *metaManager) {
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

		newDescription := strings.Join(args[1:], " ")

		world.items.changeItem(itemId, func(i *item) {
			switch it := (*i).(type) {
			case genericItem:
				it.brief = newDescription
				*i = it
				c.Write([]byte("The " + (*i).Name() + " seems less ugly than it was.\n"))
			case npc:
				it.brief = newDescription
				*i = it
				c.Write([]byte("The " + (*i).Name() + " shimmers for a minute, looking strangely different after.\n"))
			default:
				c.Write([]byte("The " + (*i).Name() + " resists your attempt to describe it."))
				fmt.Println("describe called with unknown item type '" + (*i).Id().String() + "'")
				return
			}
		})
	}
	commands["di"] = commands["describeitem"]
}

func initCommandsDirections() {
	commands["south"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(south, c, player, world)
	}
	commands["s"] = commands["south"]
	commands["north"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(north, c, player, world)
	}
	commands["n"] = commands["north"]
	commands["east"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(east, c, player, world)
	}
	commands["e"] = commands["east"]
	commands["west"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(west, c, player, world)
	}
	commands["w"] = commands["west"]
	commands["northeast"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(northeast, c, player, world)
	}
	commands["ne"] = commands["northeast"]
	commands["northwest"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(northwest, c, player, world)
	}
	commands["nw"] = commands["northwest"]
	commands["southeast"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(southeast, c, player, world)
	}
	commands["se"] = commands["southeast"]
	commands["southwest"] = func(args []string, c net.Conn, player string, world *metaManager) {
		walk(southwest, c, player, world)
	}
	commands["sw"] = commands["southwest"]
}

func initCommandsItems() {
	commands["get"] = func(args []string, c net.Conn, player string, world *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to get?\n"))
			return
		}

		realPlayer, exists := world.players.getPlayer(player)
		if !exists {
			fmt.Println("get called with invalid player2 '" + player + "'")
			return
		}

		roomId, exists := world.playerLocations.playerRoom(player)
		if !exists {
			fmt.Println("get called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("get called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		itemInt, err := strconv.Atoi(args[0])
		if err != nil {
			// getting by name, not id
			items := world.itemLocations.locationItems(identifier(roomId), ilRoom)
			found := false
			for _, itemId := range items {
				it, exists := world.items.getItem(itemId)
				if !exists {
					fmt.Println("get got nonexistent item from itemLocationManager '" + itemId.String() + "'")
				}
				if it.Name() == args[0] {
					itemInt = int(it.Id())
					found = true
					break
				}
			}
			if !found {
				c.Write([]byte("That is not here.0\n"))
				return
			}
		}
		fmt.Println("debug got " + strconv.Itoa(itemInt))
		it, exists := world.items.getItem(itemIdentifier(itemInt))
		if !exists {
			c.Write([]byte("That does not exist.\n"))
			return
		}

		switch it.(type) {
		case genericItem:
			world.itemLocations.moveItem(c, it.Id(), identifier(currentRoom.id), ilRoom, realPlayer.id, ilPlayer, func(success bool) {
				if success {
					c.Write([]byte("You pick up " + it.Brief() + ".\n"))
				} else {
					c.Write([]byte("That is not here.1\n"))
				}
			})
		case npc:
			c.Write([]byte(it.Brief() + " <stares at you awkwardly.\n"))
		}
	}
	commands["g"] = commands["get"]
	commands["drop"] = func(args []string, c net.Conn, player string, world *metaManager) {
		if len(args) < 1 {
			c.Write([]byte("What do you want to drop?\n"))
			return
		}

		roomId, exists := world.playerLocations.playerRoom(player)
		if !exists {
			fmt.Println("drop called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("drop called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		realPlayer, exists := world.players.getPlayer(player)
		if !exists {
			fmt.Println("Drop called with invalid player2 '" + player + "'")
			return
		}

		itemInt, err := strconv.Atoi(args[0])
		if err != nil {
			// getting by name, not id
			items := world.itemLocations.locationItems(identifier(realPlayer.id), ilPlayer)
			for _, itemId := range items {
				it, exists := world.items.getItem(itemId)
				if !exists {
					fmt.Println("drop got nonexistent item from itemLocationManager '" + itemId.String() + "'")
				}
				if it.Name() == args[0] {
					world.itemLocations.moveItem(c, it.Id(), realPlayer.id, ilPlayer, identifier(currentRoom.id), ilRoom, func(success bool) {
						if success {
							c.Write([]byte("You drop " + it.Brief() + ".\n"))
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

		it, exists := world.items.getItem(itemIdentifier(itemInt))
		if !exists {
			c.Write([]byte("That does not exist.\n"))
			return
		}

		world.itemLocations.moveItem(c, it.Id(), realPlayer.id, ilPlayer, identifier(currentRoom.id), ilRoom, func(success bool) {
			if success {
				c.Write([]byte("You drop " + it.Brief() + ".\n"))
			} else {
				c.Write([]byte("You aren't holding that.\n"))
			}
		})
	}

	commands["items"] = func(args []string, c net.Conn, player string, world *metaManager) {
		realPlayer, exists := world.players.getPlayer(player)
		if !exists {
			fmt.Println("Drop called with invalid player2 '" + player + "'")
			return
		}

		items := world.itemLocations.locationItems(realPlayer.id, ilPlayer)
		for _, itemId := range items {
			it, exists := world.items.getItem(itemId)
			if !exists {
				fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			c.Write([]byte(it.Id().String() + it.Name() + "\n"))
		}
	}
	commands["ii"] = commands["items"]

	commands["inventory"] = func(args []string, c net.Conn, player string, world *metaManager) {
		realPlayer, exists := world.players.getPlayer(player)
		if !exists {
			fmt.Println("inventory called with invalid player2 '" + player + "'")
			return
		}
		items := world.itemLocations.locationItems(realPlayer.id, ilPlayer)
		if len(items) == 0 {
			c.Write([]byte("You aren't carrying anything.\n"))
			return
		}
		s := "You are carrying "
		if len(items) == 1 {
			it, exists := world.items.getItem(items[0])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[0].String() + "'")
				s += "nothing of interest."
				c.Write([]byte(s))
				return
			}
			s += it.Brief()
			s += ".\n"
			c.Write([]byte(s))
			return
		}
		if len(items) == 2 {
			it, exists := world.items.getItem(items[0])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[0].String() + "'")
				s += "nothing of interest"
			} else {
				s += it.Brief()
			}
			s += ", "
			it, exists = world.items.getItem(items[1])
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[1].String() + "'")
				s += "nothing of interest"
			} else {
				s += it.Brief()
			}
			s += ".\n"
			c.Write([]byte(s))
			return
		}

		lastItem := items[len(items)-1]
		items = items[:len(items)-1]
		for _, itemId := range items {
			it, exists := world.items.getItem(itemId)
			if !exists {
				fmt.Println("inventory got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			s += it.Brief() + ", "
		}
		it, exists := world.items.getItem(lastItem)
		if !exists {
			fmt.Println("inventory got nonexistent item from itemLocationManager '" + lastItem.String() + "'")
			s += "nothing of interest"
		} else {
			s += it.Brief()
		}
		s += ".\n"
		c.Write([]byte(s))
	}
	commands["inv"] = commands["inventory"]
	commands["i"] = commands["inventory"]

	commands["itemshere"] = func(args []string, c net.Conn, player string, world *metaManager) {
		roomId, exists := world.playerLocations.playerRoom(player)
		if !exists {
			fmt.Println("items called with invalid player '" + player + "'")
			return
		}

		currentRoom, exists := world.rooms.getRoom(roomId)
		if !exists {
			fmt.Println("items called with player with invalid room '" + player + "' " + strconv.Itoa(int(roomId)))
			return
		}

		items := world.itemLocations.locationItems(identifier(currentRoom.id), ilRoom)
		for _, itemId := range items {
			it, exists := world.items.getItem(itemId)
			if !exists {
				fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			c.Write([]byte(it.Id().String() + it.Name() + "\t" + it.Brief() + "\n"))
		}
	}
	commands["ih"] = commands["itemshere"]

}

func initCommandsBasic() {
	commands["look"] = func(args []string, c net.Conn, player string, world *metaManager) {
		look(c, player, world)
	}
	commands["l"] = commands["look"]

	commands["quicklook"] = func(args []string, c net.Conn, player string, world *metaManager) {
		quicklook(c, player, world)
	}
	commands["ql"] = commands["quicklook"]

	commands["say"] = func(args []string, c net.Conn, player string, world *metaManager) {
		say(strings.Join(args, " "), player, world)
	}
	commands["'"] = commands["say"]
}

func initCommands() {
	initCommandsBasic()
	initCommandsDirections()
	initCommandsItems()
	initCommandsAdmin()
}
