package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const commandRejectMessage = "I don't understand."

var commands = map[string]func([]string, identifier, *metaManager){}

func ToProper(player string) string {
	if len(player) == 0 {
		return player
	}
	if len(player) == 1 {
		return strings.ToUpper(player)
	}
	return strings.ToUpper(string(player[0])) + player[1:]
}

func ToSentence(s string) string {
	sentenceEnd, err := regexp.Compile(`[.!?]$`) // @todo make this locale aware.
	if err != nil {
		fmt.Println(err)
		return ""
	}
	if !sentenceEnd.Match([]byte(s)) {
		s += "."
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func say(message string, playerId identifier, world *metaManager) {
	if len(message) == 0 {
		return
	}
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("say error: getPlayer got nonexistent player " + playerId.String())
		return
	}
	rId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("say error: playerRoom got nonexistent room for " + player.Name())
		return
	}
	r, exists := world.rooms.getRoom(rId)
	if !exists {
		fmt.Println("say error: getRoom got nonexistent room " + rId.String())
		return
	}
	message = ToSentence(message)
	roomMessage := Pink + ToProper(player.Name()) + " says, \"" + message + "\"" + Reset // @todo make this locale aware, << >> vs " " vs ' '
	selfMessage := Pink + "You say, \"" + message + "\"" + Reset
	go r.Write(roomMessage, world.playerLocations, player.Name())
	go player.Write(selfMessage)
}

func tell(message string, playerId identifier, telleePlayer string, world *metaManager) {
	if len(message) == 0 {
		return
	}
	telleePlayer = strings.ToLower(telleePlayer)
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("say error: getPlayer got nonexistent player " + playerId.String())
		return
	}

	if player.Name() == telleePlayer {
		go player.Write("Your own voice reverberates in your head.")
		return
	}

	tellee, exists := world.players.getPlayer(telleePlayer)
	if !exists {
		go player.Write("Your own voice reverberates in your head.")
		return
	}

	message = ToSentence(message)
	telleeMessage := Cyan + ToProper(player.Name()) + " tells you, \"" + message + "\"" + Reset // @todo make this locale aware, << >> vs " " vs ' '
	tellerMessage := Cyan + "You tell " + ToProper(telleePlayer) + ", \"" + message + "\"" + Reset
	go tellee.Write(telleeMessage)
	go player.Write(tellerMessage)
}

func walk(d Direction, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("walk called with invalid player id '" + playerId.String() + "'")
		return
	}
	world.playerLocations.movePlayer(playerId, d, func(success bool) {
		if !success {
			player.Write("You can't go there.") // @todo tell the user why (no exit, blocked, etc.)
			return
		}
		go look(playerId, world)
	})
}

func look(playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("look called with invalid player id '" + playerId.String() + "'")
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("look called with invalid  player'" + player.Name() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("look called with player with invalid room '" + player.Name() + "' " + strconv.Itoa(int(roomId)))
		return
	}
	player.Write(currentRoom.Print(world, player.Name()))
}

func quicklook(playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("quicklook called with invalid player " + playerId.String())
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("quicklook called with invalid player  '" + player.Name() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("quicklook called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}
	player.Write(currentRoom.PrintBrief(world, player.Name()))
}

func makeRoom(direction Direction, name string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("makeroom called with nonexistent player " + playerId.String())
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("makeroom called with invalid player '" + player.Name() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("makeroom called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}
	currentRoom.NewRoom(world.rooms, direction, name, "")
	player.Write(name + " materializes to the " + direction.String() + ". It is nondescript and seems as though it might fade away at any moment.")
}

func connectRoom(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("makeroom called with nonexistent player " + playerId.String())
		return
	}
	if len(args) < 2 {
		player.Write(commandRejectMessage + "5")
		return
	}
	toConnectRoomIdInt, err := strconv.Atoi(args[1])
	if err != nil {
		player.Write(commandRejectMessage + "6")
		return
	}
	toConnectRoomId := roomIdentifier(toConnectRoomIdInt)
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("connectroom called with invalid player '" + player.Name() + "'")
		return
	}

	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("connectroom called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}

	newRoomDirection := stringToDirection(args[0])
	toConnectRoom, connectionRoomExists := world.rooms.getRoom(toConnectRoomId)
	if !connectionRoomExists {
		player.Write("No room exists with the given id.")
		return
	}

	world.rooms.changeRoom(currentRoom.id, func(r *room) {
		r.exits[newRoomDirection] = toConnectRoom.id
	})
	world.rooms.changeRoom(toConnectRoom.id, func(r *room) {
		r.exits[newRoomDirection.reverse()] = currentRoom.id
		go player.Write("You become aware of a " + newRoomDirection.String() + " passage to " + toConnectRoom.name + ".")
	})
}

func describeRoom(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("describeroom called with nonexistent player " + playerId.String())
		return
	}
	if len(args) < 1 {
		player.Write(commandRejectMessage + "3") ///< @todo give better  error
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("describeroom called with invalid player '" + player.Name() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("connectroom called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}
	world.rooms.changeRoom(currentRoom.id, func(r *room) {
		r.description = strings.Join(args[0:], " ")
		go player.Write("Everything seems a bit more corporeal.")
	})
}

func roomId(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("roomid called with nonexistent player " + playerId.String())
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("roomid called with invalid player '" + player.Name() + "'")
		return
	}

	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("connectroom called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}
	player.Write(currentRoom.id.String())
}

func createItem(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("createitem called with nonexistent player " + playerId.String())
		return
	}
	if len(args) < 1 {
		player.Write("What do you want to create?")
		return
	}
	itemName := args[0]
	var it item = genericItem{
		id:    itemIdentifier(invalidIdentifier),
		name:  itemName,
		brief: "An amorphous blob"}

	id := world.items.createItem(it)
	world.itemLocations.addItem(id, player.Id(), ilPlayer)
	player.Write("A " + itemName + " materialises in your hands.")
}

func createNpc(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("createnpc called with invalid player " + playerId.String())
		return
	}
	if len(args) < 1 {
		player.Write("What do you want to create?")
		return
	}
	itemName := args[0]
	var it item = npc{
		id:       itemIdentifier(invalidIdentifier),
		name:     itemName,
		brief:    "A mysterious figure",
		sleeping: false,
		dna:      ""}
	id := world.items.createItem(it)
	world.itemLocations.addItem(id, identifier(player.id), ilPlayer)
	player.Write("A " + itemName + " materialises in your hands.")
}

func animate(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("animate called with invalid player " + playerId.String())
		return
	}
	if len(args) < 2 {
		player.Write("Who do you want to animate?")
		return
	}

	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		player.Write("Please provide a valid identifier to animate.")
		return
	}
	itemId := itemIdentifier(itemInt)

	newDna := strings.Join(args[1:], " ")

	world.items.changeItem(itemId, func(i *item) {
		switch it := (*i).(type) {
		case genericItem:
			go player.Write("You cannot animate items.")
		case npc:
			it.dna = newDna
			*i = it
			go player.Write((*i).Brief() + " suddenly comes to life.")
		default:
			player.Write("The " + (*i).Name() + " resists your attempt to animate it.")
			go fmt.Println("describe called with unknown item type '" + (*i).Id().String() + "'")
		}
	})
}

func describeItem(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("describeItem called with invalid player " + playerId.String())
		return
	}
	if len(args) < 2 {
		player.Write("What do you want to describe?")
		return
	}

	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		player.Write("Please provide a valid identifier to describe.")
		return
	}
	itemId := itemIdentifier(itemInt)

	newDescription := strings.Join(args[1:], " ")

	world.items.changeItem(itemId, func(i *item) {
		switch it := (*i).(type) {
		case genericItem:
			it.brief = newDescription
			*i = it
			player.Write("The " + (*i).Name() + " seems less ugly than it was.")
		case npc:
			it.brief = newDescription
			*i = it
			player.Write("The " + (*i).Name() + " shimmers for a minute, looking strangely different after.")
		default:
			player.Write("The " + (*i).Name() + " resists your attempt to describe it.")
			fmt.Println("describe called with unknown item type '" + (*i).Id().String() + "'")
			return
		}
	})
}

/// @todo fix this to lock the player's location before moving the item
func get(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("get called with invalid player " + playerId.String())
		return
	}
	if len(args) < 1 {
		player.Write("What do you want to get?")
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("get called with invalid player '" + player.Name() + "'")
		return
	}

	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("get called with player with invalid room '" + player.Name() + "' " + roomId.String())
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
			player.Write("That is not here.0")
			return
		}
	}
	it, exists := world.items.getItem(itemIdentifier(itemInt))
	if !exists {
		player.Write("That does not exist.")
		return
	}

	switch it.(type) {
	case genericItem:
		world.itemLocations.moveItem(it.Id(), identifier(currentRoom.id), ilRoom, player.Id(), ilPlayer, func(success bool) {
			if success {
				player.Write("You pick up " + it.Brief() + ".")
			} else {
				player.Write("That is not here.1")
			}
		})
	case npc:
		player.Write(it.Brief() + " stares at you awkwardly.")
	}
}

/// @todo fix this to lock the player's location before moving the item
func drop(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("get called with invalid player " + playerId.String())
		return
	}

	if len(args) < 1 {
		player.Write("What do you want to drop?")
		return
	}

	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("drop called with invalid player '" + player.Name() + "'")
		return
	}

	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("drop called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}
	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		// getting by name, not id
		items := world.itemLocations.locationItems(playerId, ilPlayer)
		found := false
		for _, itemId := range items {
			it, exists := world.items.getItem(itemId)
			if !exists {
				fmt.Println("drop got nonexistent item from itemLocationManager '" + itemId.String() + "'")
			}
			if it.Name() == args[0] {
				itemInt = int(itemId)
				found = true
				break
			}
		}
		if !found {
			player.Write("You are not holding that.")
			return
		}
	}

	it, exists := world.items.getItem(itemIdentifier(itemInt))
	if !exists {
		player.Write("That does not exist.")
		return
	}
	world.itemLocations.moveItem(it.Id(), playerId, ilPlayer, identifier(currentRoom.id), ilRoom, func(success bool) {
		if success {
			player.Write("You drop " + it.Brief() + ".")
			fmt.Println("dropping...")
			if npc, ok := it.(npc); ok {
				npc.Animate(world) ///< @todo this probably should be in a Manager, not Commands. Animate objects?
			}
		} else {
			player.Write("You aren't holding that.")
		}
	})
}

func items(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("Items called with invalid player2 '" + playerId.String() + "'")
		return
	}

	items := world.itemLocations.locationItems(playerId, ilPlayer)
	itemString := ""
	for _, itemId := range items {
		it, exists := world.items.getItem(itemId)
		if !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
		}
		itemString += it.Id().String() + it.Name() + "\r\n"
	}
	if len(itemString) > 0 {
		player.Write(itemString[:len(itemString)-2])
	}
}

func itemsHere(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("itemshere called with invalid player2 '" + playerId.String() + "'")
		return
	}
	roomId, exists := world.playerLocations.playerRoom(player.Name())
	if !exists {
		fmt.Println("items called with invalid player '" + player.Name() + "'")
		return
	}

	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("items called with player with invalid room '" + player.Name() + "' " + roomId.String())
		return
	}

	items := world.itemLocations.locationItems(identifier(currentRoom.id), ilRoom)
	for _, itemId := range items {
		it, exists := world.items.getItem(itemId)
		if !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
		}
		player.Write(it.Id().String() + it.Name() + "\t" + it.Brief())
	}

}

func inventory(args []string, playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("inventory called with invalid player2 '" + playerId.String() + "'")
		return
	}
	items := world.itemLocations.locationItems(playerId, ilPlayer)
	if len(items) == 0 {
		player.Write("You aren't carrying anything.")
		return
	}
	s := "You are carrying "
	if len(items) == 1 {
		it, exists := world.items.getItem(items[0])
		if !exists {
			fmt.Println("inventory got nonexistent item from itemLocationManager '" + items[0].String() + "'")
			s += "nothing of interest."
			player.Write(s)
			return
		}
		s += it.Brief()
		s += "."
		player.Write(s)
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
		s += "."
		player.Write(s)
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
	s += "."
	player.Write(s)
}

func help(playerId identifier, world *metaManager) {
	player, exists := world.players.getPlayerById(playerId)
	if !exists {
		fmt.Println("help called with invalid player'" + playerId.String() + "'")
		return
	}
	s := `movement
------------------------------
To move in a direction, simply type the cardinal direction you wish to move in, e.g. "north". Shortcuts also work, e.g. "n".


command		brief	syntax
------------------------------
say			say message
tell			tell person message
look		l	look
quicklook	ql	quicklook
makeroom	mr	makeroom direction title
connectroom	cr	connectroom direction roomId
describeroom	dr	describeroom description
roomid			roomid
createitem	ci	creatitem name
createnpc	cn	createnpc name
describeitem	di	describeitem itemId description
describenpc	dn	describenpc npcId description
animate		an	animate npcId script
get		g	get itemId/itemName
drop			drop itemId/itemName
items		ii	items
itemshere	ih	itemshere
inventory	i	inventory



animating
------------------------------
NPCs (non-player-characters) can be animated via javascript.

All gomud commands are newline-delimited, so you must remove all newlines from your script before passing it to animate.

For efficiency, your script should return as soon as possible. You should call mud_reval() to specify when your script will be called again, immediately before returning.

A variable named "self" is available during execution. This is the ID of the current NPC, and necessary for many hook functions.

The current available "hook" functions available in Javascript are:
--------------------------------------------------------------------------------
mud_println(text)                      print text to the server's console
mud_getPlayer(name)                    get a struct containing the player's name and ID
mud_moveRandom(self)                   move in a random direction
mud_reval(self, wait)                  execute this NPC's animation script again in *wait* milliseconds
mud_roomPlayers(self)                  get an array of the names of players in the room
mud_attackPlayer(self, player, damage) attack the given player for the given integral amount of damage
`
	player.Write(s)
}

func initCommandsAdmin() {
	commands["makeroom"] = func(args []string, playerId identifier, world *metaManager) {
		if len(args) < 2 {
			player, exists := world.players.getPlayerById(playerId)
			if !exists {
				fmt.Println("makeroom error: getPlayer got nonexistent player " + playerId.String())
				return
			}
			player.Write(commandRejectMessage + "3") ///< @todo give better error
			return
		}
		newRoomDirection := stringToDirection(args[0])
		if newRoomDirection < north || newRoomDirection > southwest {
			player, exists := world.players.getPlayerById(playerId)
			if !exists {
				fmt.Println("makeroom error: getPlayer got nonexistent player " + playerId.String())
				return
			}
			player.Write(commandRejectMessage + "4") ///< @todo give better error
			return
		}
		newRoomName := strings.Join(args[1:], " ")
		if len(newRoomName) == 0 {
			player, exists := world.players.getPlayerById(playerId)
			if !exists {
				fmt.Println("makeroom error: getPlayer got nonexistent player " + playerId.String())
				return
			}
			player.Write(commandRejectMessage + "5") ///< @todo give better error
			return
		}
		makeRoom(newRoomDirection, newRoomName, playerId, world)
	}
	commands["mr"] = commands["makeroom"]

	commands["connectroom"] = func(args []string, playerId identifier, world *metaManager) {
		connectRoom(args, playerId, world)
	}
	commands["cr"] = commands["connectroom"]

	commands["describeroom"] = func(args []string, playerId identifier, world *metaManager) {
		describeRoom(args, playerId, world)
	}
	commands["dr"] = commands["describeroom"]

	commands["roomid"] = func(args []string, playerId identifier, world *metaManager) {
		roomId(args, playerId, world)
	}

	commands["createitem"] = func(args []string, playerId identifier, world *metaManager) {
		createItem(args, playerId, world)
	}
	commands["ci"] = commands["createitem"]

	commands["createnpc"] = func(args []string, playerId identifier, world *metaManager) {
		createNpc(args, playerId, world)
	}
	commands["cn"] = commands["createnpc"]

	commands["describeitem"] = func(args []string, playerId identifier, world *metaManager) {
		describeItem(args, playerId, world)
	}
	commands["di"] = commands["describeitem"]
	commands["describenpc"] = commands["describeitem"]
	commands["dn"] = commands["describenpc"]

	commands["animate"] = func(args []string, playerId identifier, world *metaManager) {
		animate(args, playerId, world)
	}
	commands["an"] = commands["animate"]
	commands["help"] = func(args []string, playerId identifier, world *metaManager) {
		help(playerId, world)
	}
	commands["?"] = commands["help"]
}

func initCommandsDirections() {
	commands["south"] = func(args []string, playerId identifier, world *metaManager) {
		walk(south, playerId, world)
	}
	commands["s"] = commands["south"]
	commands["north"] = func(args []string, playerId identifier, world *metaManager) {
		walk(north, playerId, world)
	}
	commands["n"] = commands["north"]
	commands["east"] = func(args []string, playerId identifier, world *metaManager) {
		walk(east, playerId, world)
	}
	commands["e"] = commands["east"]
	commands["west"] = func(args []string, playerId identifier, world *metaManager) {
		walk(west, playerId, world)
	}
	commands["w"] = commands["west"]
	commands["northeast"] = func(args []string, playerId identifier, world *metaManager) {
		walk(northeast, playerId, world)
	}
	commands["ne"] = commands["northeast"]
	commands["northwest"] = func(args []string, playerId identifier, world *metaManager) {
		walk(northwest, playerId, world)
	}
	commands["nw"] = commands["northwest"]
	commands["southeast"] = func(args []string, playerId identifier, world *metaManager) {
		walk(southeast, playerId, world)
	}
	commands["se"] = commands["southeast"]
	commands["southwest"] = func(args []string, playerId identifier, world *metaManager) {
		walk(southwest, playerId, world)
	}
	commands["sw"] = commands["southwest"]
}

func initCommandsItems() {
	commands["get"] = func(args []string, playerId identifier, world *metaManager) {
		get(args, playerId, world)
	}
	commands["g"] = commands["get"]
	commands["drop"] = func(args []string, playerId identifier, world *metaManager) {
		drop(args, playerId, world)
	}

	commands["items"] = func(args []string, playerId identifier, world *metaManager) {
		items(args, playerId, world)
	}
	commands["ii"] = commands["items"]

	commands["inventory"] = func(args []string, playerId identifier, world *metaManager) {
		inventory(args, playerId, world)
	}
	commands["inv"] = commands["inventory"]
	commands["i"] = commands["inventory"]

	commands["itemshere"] = func(args []string, playerId identifier, world *metaManager) {
		itemsHere(args, playerId, world)
	}
	commands["ih"] = commands["itemshere"]

}

func initCommandsBasic() {
	commands["look"] = func(args []string, playerId identifier, world *metaManager) {
		look(playerId, world)
	}
	commands["l"] = commands["look"]

	commands["quicklook"] = func(args []string, playerId identifier, world *metaManager) {
		quicklook(playerId, world)
	}
	commands["ql"] = commands["quicklook"]

	commands["say"] = func(args []string, playerId identifier, world *metaManager) {
		say(strings.Join(args, " "), playerId, world)
	}
	commands["'"] = commands["say"]

	commands["tell"] = func(args []string, playerId identifier, world *metaManager) {
		if len(args) < 2 {
			return
		}
		tellee := args[0]
		args = args[1:]
		tell(strings.Join(args, " "), playerId, tellee, world)
	}
}

func initCommands() {
	initCommandsBasic()
	initCommandsDirections()
	initCommandsItems()
	initCommandsAdmin()
}
