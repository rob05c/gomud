/*
commands.go handles the routing of commands.

When a player types a command, the cooresponding function is called from the commands map

Note Commands are the primary place the "chain locking" pattern is used.
If you get more than 1 setter at once, you MUST use this pattern to prevent deadlock and starvation.
See the thing.go file comment for more details.
Look at functions in this file which use NextChainTime for examples.
*/
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const commandRejectMessage = "I don't understand."

type CommandFunc func([]string, identifier, *World)

var commands = map[string]CommandFunc{}

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

func tryPlayerWrite(playerId identifier, players *PlayerManager, message string, error string) bool {
	player, exists := players.GetById(playerId)
	if !exists {
		fmt.Println(error + " " + playerId.String())
		return false
	}
	player.Write(message)
	return true
}

func say(args []string, playerId identifier, world *World) {
	sayMsg(strings.Join(args, " "), playerId, world)
}

func sayMsg(message string, playerId identifier, world *World) {
	if len(message) == 0 {
		return
	}
	player, exists := PlayerManager(*world.players).GetById(playerId)
	if !exists {
		fmt.Println("say error: getPlayer got nonexistent player " + playerId.String())
		return
	}
	rid := player.Room

	r, ok := RoomManager(*world.rooms).GetById(rid)
	if !ok {
		return // don't print error - RoomManager will
	}

	message = ToSentence(message)
	RoomMessage := Pink + ToProper(player.Name()) + " says, \"" + message + "\"" + Reset // @todo make this locale aware, << >> vs " " vs ' '
	selfMessage := Pink + "You say, \"" + message + "\"" + Reset
	go r.Write(RoomMessage, *world.players, player.Name())
	go player.Write(selfMessage)
}

func tell(args []string, playerId identifier, world *World) {
	if len(args) < 2 {
		return
	}
	tellee := args[0]
	args = args[1:]
	tellMsg(strings.Join(args, " "), playerId, tellee, world)
}

func tellMsg(message string, playerId identifier, telleePlayer string, world *World) {
	if len(message) == 0 {
		return
	}
	telleePlayer = strings.ToLower(telleePlayer)
	player, exists := world.players.GetById(playerId)
	if !exists {
		fmt.Println("say error: getPlayer got nonexistent player " + playerId.String())
		return
	}

	if player.Name() == telleePlayer {
		go player.Write("Your own voice reverberates in your head.")
		return
	}

	tellee, exists := world.players.GetByName(telleePlayer)
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

func walkSouth(args []string, playerId identifier, world *World) {
	walk(south, playerId, world)
}

func walkNorth(args []string, playerId identifier, world *World) {
	walk(north, playerId, world)
}

func walkEast(args []string, playerId identifier, world *World) {
	walk(west, playerId, world)
}

func walkWest(args []string, playerId identifier, world *World) {
	walk(west, playerId, world)
}

func walkNortheast(args []string, playerId identifier, world *World) {
	walk(northeast, playerId, world)
}

func walkNorthwest(args []string, playerId identifier, world *World) {
	walk(northwest, playerId, world)
}

func walkSoutheast(args []string, playerId identifier, world *World) {
	walk(southeast, playerId, world)
}

func walkSouthwest(args []string, playerId identifier, world *World) {
	walk(southwest, playerId, world)
}

func walk(d Direction, playerId identifier, world *World) {
	_, exists := world.players.GetById(playerId)
	if !exists {
		fmt.Println("walk called with invalid player id '" + playerId.String() + "'")
		return
	}
	world.players.Move(playerId, d, world)
}

func look(args []string, playerId identifier, world *World) {
	player, exists := world.players.GetById(playerId)
	if !exists {
		fmt.Println("look called with invalid player id '" + playerId.String() + "'")
		return
	}
	RoomId := player.Room
	ok := RoomManager(*world.rooms).ChangeById(identifier(RoomId), func(r *Room) {
		player.Write(r.Print(world, player.Name()))
	})
	if !ok {
		fmt.Println("look called with player with invalid Room '" + player.Name() + "' " + strconv.Itoa(int(RoomId)))
	}
}

func quicklook(args []string, playerId identifier, world *World) {
	player, exists := world.players.GetById(playerId)
	if !exists {
		fmt.Println("quicklook called with invalid player " + playerId.String())
		return
	}
	RoomId := player.Room
	ok := RoomManager(*world.rooms).ChangeById(identifier(RoomId), func(r *Room) {
		player.Write(r.PrintBrief(world, player.Name()))
	})
	if !ok {
		fmt.Println("quicklook called with player with invalid Room '" + player.Name() + "' " + RoomId.String())
		return
	}
}

func makeRoom(direction Direction, name string, playerId identifier, world *World) {
	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		things := make([]SetterMsg, 0, 2)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("makeRoom error: player chan closed " + playerId.String())
			return
		} else if resetChain {
			continue
		}
		things = append(things, playerSet)

		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(playerSet.it.(*Player).Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: room chan closed " + playerId.String())
			ReleaseThings(things)
			return
		} else if resetChain {
			ReleaseThings(things)
			continue
		}
		things = append(things, roomSet)

		newRoom := Room{
			name:        name,
			Description: "",
			Exits:       make(map[Direction]identifier),
			Players:     make(map[identifier]bool),
			Items:       make(map[identifier]PlayerItemType),
		}
		newRoom.Exits[direction.reverse()] = roomSet.it.Id()
		newRoomId := ThingManager(*world.rooms).Add(&newRoom)
		roomSet.it.(*Room).Exits[direction] = newRoomId
		playerSet.it.(*Player).Write(name + " materializes to the " + direction.String() + ". It is nondescript and seems as though it might fade away at any moment.")
		ReleaseThings(things)
		break
	}
}

func connectRoom(args []string, playerId identifier, world *World) {
	chainTime := <-NextChainTime
	if len(args) < 2 {
		tryPlayerWrite(playerId, world.players, "What do you want to connect?", "connectroom error: insufficient args and no player")
		return // false
	}
	toConnectRoomIdInt, err := strconv.Atoi(args[1])
	if err != nil {
		tryPlayerWrite(playerId, world.players, "What do you want to connect?", "connectroom error: invalid roomid and no player")
		return // false
	}
	toConnectRoomId := identifier(toConnectRoomIdInt)
	connectRoomAccessor := ThingManager(*world.rooms).GetThingAccessor(toConnectRoomId)

	newRoomDirection := stringToDirection(args[0])
	if newRoomDirection == -1 {
		tryPlayerWrite(playerId, world.players, "What direction do you want to connect?", "connectroom error: invalid direction and no player")
		return // false
	}

	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		sets := make([]SetterMsg, 0, 3)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("connectRoom error: player chan closed " + playerId.String())
			return // false
		} else if resetChain {
			continue
		}
		sets = append(sets, playerSet)
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(playerSet.it.(*Player).Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: room chan closed " + playerId.String())
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)
		connectRoomSet, ok, resetChain := connectRoomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: room chan closed " + playerId.String())
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, connectRoomSet)

		roomSet.it.(*Room).Exits[newRoomDirection] = connectRoomSet.it.Id()
		connectRoomSet.it.(*Room).Exits[newRoomDirection.reverse()] = roomSet.it.Id()
		playerSet.it.(*Player).Write("You become aware of a " + newRoomDirection.String() + " passage to " + connectRoomSet.it.Name() + ".")
		ReleaseThings(sets)
		break
	}
	return // true
}

func describeRoom(args []string, playerId identifier, world *World) {
	if len(args) < 1 {
		tryPlayerWrite(playerId, world.players, commandRejectMessage, "describeRoom called with invalid player")
		return // false
	}

	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		sets := make([]SetterMsg, 0, 2)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("describeroom error: player chan closed " + playerId.String())
			return // false
		} else if resetChain {
			continue
		}
		sets = append(sets, playerSet)
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(playerSet.it.(*Player).Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("describeroom error: room chan closed " + playerId.String())
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)

		r := roomSet.it.(*Room)
		r.Description = strings.Join(args[0:], " ")
		roomSet.it = r
		playerSet.it.(*Player).Write("Everything seems a bit more corporeal.")
		ReleaseThings(sets)
		break
	}
	return // true
}

func RoomId(args []string, playerId identifier, world *World) {
	player, exists := world.players.GetById(playerId)
	if !exists {
		fmt.Println("Roomid called with nonexistent player " + playerId.String())
		return
	}
	currentRoom, exists := RoomManager(*world.rooms).GetById(player.Room)
	if !exists {
		fmt.Println("connectRoom called with player with invalid Room '" + player.Name() + "' " + player.Room.String())
		return
	}
	player.Write(currentRoom.id.String())
}

func createItem(args []string, playerId identifier, world *World) {
	if len(args) < 3 {
		tryPlayerWrite(playerId, world.players, "A new item must have a name and at least a 2-word description", "createItem called with invalid player")
		return
	}
	item := Item{
		id:           invalidIdentifier,
		name:         args[0],
		brief:        strings.Join(args[1:], " "),
		Location:     playerId,
		LocationType: ilPlayer,
		Items:        make(map[identifier]bool),
	}
	id := ThingManager(*world.items).Add(&item)
	world.players.ChangeById(playerId, func(player *Player) {
		player.Items[id] = piItem
		tryPlayerWrite(playerId, world.players, "A "+item.Name()+" materialises in your hands.", "createItem succeeded but player disappeared")
	})
}

func createNpc(args []string, playerId identifier, world *World) {
	if len(args) < 1 {
		tryPlayerWrite(playerId, world.players, "Who do you want to create?", "createNpc called with invalid player")
		return
	}
	npc := Npc{
		id:           invalidIdentifier,
		name:         args[0],
		Brief:        "A mysterious figure",
		Location:     playerId,
		LocationType: ilPlayer,
		Dna:          "",
		Sleeping:     false,
		Items:        make(map[identifier]bool),
	}
	id := ThingManager(*world.npcs).Add(&npc)
	world.players.ChangeById(playerId, func(player *Player) {
		player.Items[id] = piNpc
		tryPlayerWrite(playerId, world.players, "A "+npc.Name()+" materialises in your hands.", "createNpc succeeded but player disappeared")
	})
}

func animate(args []string, playerId identifier, world *World) {
	if len(args) < 2 {
		tryPlayerWrite(playerId, world.players, "Who do you want to animate?", "animate called with invalid player")
		return
	}

	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		tryPlayerWrite(playerId, world.players, "Please provide a valid identifier to animate", "animate called with invalid id")
		return
	}
	itemId := identifier(itemInt)
	newDna := strings.Join(args[1:], " ")

	world.npcs.ChangeById(itemId, func(n *Npc) {
		n.Dna = newDna
		tryPlayerWrite(playerId, world.players, n.Brief+" suddenly comes to life.", "animate succeeded but player disappeared")
	})
}

func describeNpc(args []string, playerId identifier, world *World) {
	if len(args) < 2 {
		tryPlayerWrite(playerId, world.players, "What do you want to describe?", "describeNpc called with invalid params")
		return
	}

	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		tryPlayerWrite(playerId, world.players, "Please provide a valid identifier to describe?", "describeNpc called with invalid id")
		return
	}
	itemId := identifier(itemInt)
	newDescription := strings.Join(args[1:], " ")

	world.npcs.ChangeById(itemId, func(n *Npc) {
		n.Brief = newDescription
		tryPlayerWrite(playerId, world.players, "The "+n.Name()+" shimmers for a minute, looking strangely different after.", "describeNpc succeeded but player disappered")
	})
}

func describeItem(args []string, playerId identifier, world *World) {
	if len(args) < 2 {
		tryPlayerWrite(playerId, world.players, "What do you want to describe?", "describeItem called with invalid params")
		return
	}

	itemInt, err := strconv.Atoi(args[0])
	if err != nil {
		tryPlayerWrite(playerId, world.players, "Please provide a valid identifier to describe?", "describeItem called with invalid id")
		return
	}
	itemId := identifier(itemInt)
	newDescription := strings.Join(args[1:], " ")

	world.items.ChangeById(itemId, func(i *Item) {
		i.brief = newDescription
		tryPlayerWrite(playerId, world.players, "The "+i.Name()+" shimmers for a minute, looking strangely different after.", "describeItem succeeded but player disappered")
	})
}

func get(args []string, playerId identifier, world *World) {
	const notHereMsg = "That is not here."
	const cantGetMsg = "You can't pick that up."
	if len(args) < 1 {
		tryPlayerWrite(playerId, world.players, "What do you want to get?", "get called with invalid params")
		return // false
	}

	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		sets := make([]SetterMsg, 0, 2)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("get error: player chan closed " + playerId.String())
			return // false
		} else if resetChain {
			//			fmt.Println("chain prempted: looping " + playerId.String())
			continue
		}
		sets = append(sets, playerSet)
		player := playerSet.it.(*Player)
		roomId := player.Room
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(roomId)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("get error: room chan closed " + player.Id().String() + " room " + roomId.String())
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			//			fmt.Println("chain prempted: looping " + playerId.String())
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)

		itemInt, err := strconv.Atoi(args[0])
		room := roomSet.it.(*Room)
		if err == nil {
			itemId := identifier(itemInt)
			itemType, ok := room.Items[itemId]
			if !ok {
				tryPlayerWrite(playerId, world.players, notHereMsg, "get called with invalid id")
				ReleaseThings(sets)
				return // false
			} else if itemType != piItem {
				tryPlayerWrite(playerId, world.players, cantGetMsg, "get called with nonitem")
				ReleaseThings(sets)
				return // false
			}
		} else {
			found := false
			npcFound := false
			for itemId, itemType := range room.Items {
				if npcFound == false && itemType == piNpc {
					_, npcFound = world.npcs.GetById(itemId)
					continue
				}
				if itemType != piItem {
					continue
				} else if it, exists := world.items.GetById(itemId); !exists {
					fmt.Println("get got nonexistent item in room '" + itemId.String() + "'")
					continue
				} else if it.Name() == args[0] {
					itemInt = int(it.Id())
					found = true
					break
				}
			}
			if !found {
				if npcFound {
					tryPlayerWrite(playerId, world.players, cantGetMsg, "get player failed to write")
				} else {
					tryPlayerWrite(playerId, world.players, notHereMsg, "get player failed to write")
				}
				ReleaseThings(sets)
				return // false
			}
		}

		itemAccessor := ThingManager(*world.items).GetThingAccessor(identifier(itemInt))
		itemSet, ok, resetChain := itemAccessor.TryGet(chainTime)
		if !ok {
			tryPlayerWrite(playerId, world.players, notHereMsg, "get error: item chan closed")
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, itemSet)

		itemId := identifier(itemInt)
		item := itemSet.it.(*Item)
		delete(room.Items, itemId)
		player.Items[itemId] = piItem
		item.Location = player.Id()
		itemSet.it = item
		player.Write("You pick up " + item.Brief())
		room.Write(ToProper(player.Name())+" picks up "+item.Brief(), *world.players, player.Name())
		ReleaseThings(sets)
		break
	}
	return // true
}

func drop(args []string, playerId identifier, world *World) {
	if len(args) < 1 {
		tryPlayerWrite(playerId, world.players, "What do you want to drop?", "drop called with invalid params")
		return // false
	}

	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		sets := make([]SetterMsg, 0, 3)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("drop error: player chan closed " + playerId.String())
			return // false
		} else if resetChain {
			//			fmt.Println("drop prempted: looping " + playerId.String())
			continue
		}
		sets = append(sets, playerSet)
		player := playerSet.it.(*Player)
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(player.Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("drop error: room chan closed " + playerId.String())
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)

		var dropType PlayerItemType = piItem
		itemInt, err := strconv.Atoi(args[0])
		if err == nil {
			itemId := identifier(itemInt)
			dropType, ok = player.Items[itemId]
			if !ok {
				tryPlayerWrite(playerId, world.players, "You aren't holding that.", "drop called with invalid id")
				ReleaseThings(sets)
				return // false
			}

		} else {
			found := false
			for itemId, _ := range player.Items {
				var it Thing
				item, exists := world.items.GetById(itemId)
				if !exists {
					npc, npcExists := world.npcs.GetById(itemId)
					if npcExists {
						exists = true
						dropType = piNpc
						it = Thing(npc)
					}
				} else {
					it = Thing(item)
				}
				if !exists {
					fmt.Println("drop got nonexistent item in player '" + itemId.String() + "'")
					continue
				} else if it.Name() == args[0] {
					itemInt = int(it.Id())
					found = true
					break
				}
			}
			if !found {
				tryPlayerWrite(playerId, world.players, "You aren't holding that.", "drop player failed to write")
				ReleaseThings(sets)
				return // false
			}
		}

		var itemAccessor ThingAccessor
		if dropType == piItem {
			itemAccessor = ThingManager(*world.items).GetThingAccessor(identifier(itemInt))
		} else {
			itemAccessor = ThingManager(*world.npcs).GetThingAccessor(identifier(itemInt))
		}
		itemSet, ok, resetChain := itemAccessor.TryGet(chainTime)
		if !ok {
			tryPlayerWrite(playerId, world.players, "You aren't holding that.", "drop error: item chan closed")
			ReleaseThings(sets)
			return // false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, itemSet)

		itemId := identifier(itemInt)
		room := roomSet.it.(*Room)
		delete(player.Items, itemId)
		room.Items[itemId] = dropType
		var itemBrief string
		if dropType == piItem {
			item := itemSet.it.(*Item)
			item.Location = room.Id()
			item.LocationType = ilRoom
			itemSet.it = item
			itemBrief = item.Brief()
		} else if dropType == piNpc {
			npc := itemSet.it.(*Npc)
			npc.Location = room.Id()
			npc.LocationType = ilRoom
			itemSet.it = npc
			itemBrief = npc.Brief
			npc.Animate(world)
		} else {
			panic("Unknown drop type (dev: you need to implement)")
		}
		player.Write("You drop " + itemBrief)
		room.Write(ToProper(player.Name())+" drops "+itemBrief, *world.players, player.Name())
		ReleaseThings(sets)
		break
	}
	return // true
}

func items(args []string, playerId identifier, world *World) {
	world.players.ChangeById(playerId, func(player *Player) {
		itemString := ""
		for itemId, itemType := range player.Items {
			var manager ThingManager
			switch itemType {
			case piItem:
				manager = ThingManager(*world.items)
			case piNpc:
				manager = ThingManager(*world.npcs)
			default:
				fmt.Println("Items got invalid item type  '" + playerId.String() + "'")
				continue
			}
			it, exists := manager.GetById(itemId)
			if !exists {
				fmt.Println("Items got invalid item  '" + playerId.String() + "'")
				continue
			}
			itemString += it.Id().String() + it.Name() + "\r\n"
		}
		if len(itemString) > 0 {
			player.Write(itemString[:len(itemString)-2])
		} else {
			player.Write("")
		}
	})
}

func itemsHere(args []string, playerId identifier, world *World) {
	if len(args) < 1 {
		tryPlayerWrite(playerId, world.players, "What do you want to drop?", "drop called with invalid params")
		return
	}

	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessor(playerId)
	for {
		sets := make([]SetterMsg, 0, 2)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("describeroom error: player chan closed " + playerId.String())
			return
		} else if resetChain {
			continue
		}
		sets = append(sets, playerSet)
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(playerSet.it.(*Player).Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("describeroom error: room chan closed " + playerId.String())
			ReleaseThings(sets)
			return
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)

		itemString := ""
		for itemId, itemType := range roomSet.it.(*Room).Items {
			var manager ThingManager
			switch itemType {
			case piItem:
				manager = ThingManager(*world.items)
			case piNpc:
				manager = ThingManager(*world.npcs)
			default:
				fmt.Println("Items got invalid item type  '" + playerId.String() + "'")
				continue
			}
			it, exists := manager.GetById(itemId)
			if !exists {
				continue
			}
			itemString += itemId.String() + it.Name() + "\r\n"
		}
		if len(itemString) > 0 {
			playerSet.it.(*Player).Write(itemString[:len(itemString)-2])
		} else {
			playerSet.it.(*Player).Write("")
		}
		ReleaseThings(sets)
		break
	}
}

func inventory(args []string, playerId identifier, world *World) {
	world.players.ChangeById(playerId, func(player *Player) {
		s := "You are carrying "
		var items []string
		for itemId, itemType := range player.Items {
			switch itemType {
			case piItem:
				it, exists := world.items.GetById(itemId)
				if !exists {
					continue
				}
				items = append(items, it.Brief())
			case piNpc:
				it, exists := world.npcs.GetById(itemId)
				if !exists {
					continue
				}
				items = append(items, it.Brief)
			default:
				fmt.Println("Items got invalid item type  '" + playerId.String() + "'")
				continue
			}
		}
		if len(items) == 0 {
			s += "nothing."
			player.Write(s)
			return
		}
		if len(items) == 1 {
			s += items[0]
			s += "."
			player.Write(s)
			return
		}
		if len(items) == 2 {
			s += items[0]
			s += ", "
			s += items[1]
			s += "."
			player.Write(s)
			return
		}

		lastItem := items[len(items)-1]
		items = items[:len(items)-1]
		for _, item := range items {
			s += item + ", "
		}
		s += lastItem
		s += "."
		player.Write(s)
	})
}

func help(args []string, playerId identifier, world *World) {
	s := "movement\r\n" +
		"------------------------------\r\n" +
		"To move in a direction, simply type the cardinal direction you wish to move in, e.g. 'north'. Shortcuts also work, e.g. 'n'.\r\n" +
		"\r\n" +
		"\r\n" +
		"command		brief	syntax\r\n" +
		"------------------------------\r\n" +
		"say			say message\r\n" +
		"tell			tell person message\r\n" +
		"look		l	look\r\n" +
		"quicklook	ql	quicklook\r\n" +
		"makeRoom	mr	makeRoom direction title\r\n" +
		"connectRoom	cr	connectRoom direction RoomId\r\n" +
		"describeRoom	dr	describeRoom description\r\n" +
		"roomid			roomid\r\n" +
		"createitem	ci	creatitem name description\r\n" +
		"createnpc	cn	createnpc name\r\n" +
		"describeitem	di	describeitem itemId description\r\n" +
		"describenpc	dn	describenpc npcId description\r\n" +
		"animate	an	animate npcId script\r\n" +
		"get		g	get itemId/itemName\r\n" +
		"drop			drop itemId/itemName\r\n" +
		"items		ii	items\r\n" +
		"itemshere	ih	itemshere\r\n" +
		"inventory	i	inventory\r\n" +
		"\r\n" +
		"\r\n" +
		"\r\n" +
		"animating\r\n" +
		"------------------------------\r\n" +
		"NPCs (non-player-characters) can be animated via javascript.\r\n" +
		"\r\n" +
		"All gomud commands are newline-delimited, so you must remove all newlines from your script before passing it to animate.\r\n" +
		"\r\n" +
		"For efficiency, your script should return as soon as possible. You should call mud_reval() to specify when your script will be called again, immediately before returning.\r\n" +
		"\r\n" +
		"A variable named 'self' is available during execution. This is the ID of the current NPC, and necessary for many hook functions.\r\n" +
		"\r\n" +
		"The current available 'hook' functions available in Javascript are:\r\n" +
		"--------------------------------------------------------------------------------\r\n" +
		"mud_println(text)                      print text to the server's console\r\n" +
		"mud_getPlayer(name)                    get a struct containing the player's name and ID\r\n" +
		"mud_moveRandom(self)                   move in a random direction\r\n" +
		"mud_reval(self, wait)                  execute this NPC's animation script again in *wait* milliseconds\r\n" +
		"mud_RoomPlayers(self)                  get an array of the names of players in the Room\r\n" +
		"mud_attackPlayer(self, player, damage) attack the given player for the given integral amount of damage\r\n"
	tryPlayerWrite(playerId, world.players, s, "help error: player chan closed")
}

func makeroom(args []string, playerId identifier, world *World) {
	if len(args) < 2 {
		player, exists := world.players.GetById(playerId)
		if !exists {
			fmt.Println("makeRoom error: getPlayer got nonexistent player " + playerId.String())
			return
		}
		player.Write(commandRejectMessage + "3") ///< @todo give better error
		return
	}
	newRoomDirection := stringToDirection(args[0])
	if newRoomDirection < north || newRoomDirection > southwest {
		player, exists := world.players.GetById(playerId)
		if !exists {
			fmt.Println("makeRoom error: getPlayer got nonexistent player " + playerId.String())
			return
		}
		player.Write(commandRejectMessage + "4") ///< @todo give better error
		return
	}
	newRoomName := strings.Join(args[1:], " ")
	if len(newRoomName) == 0 {
		player, exists := world.players.GetById(playerId)
		if !exists {
			fmt.Println("makeRoom error: getPlayer got nonexistent player " + playerId.String())
			return
		}
		player.Write(commandRejectMessage + "5") ///< @todo give better error
		return
	}
	makeRoom(newRoomDirection, newRoomName, playerId, world)
}

func initCommands() {
	commands = map[string]CommandFunc{
		// admin
		"makeroom":     makeroom,
		"mr":           makeroom,
		"connectroom":  connectRoom,
		"cr":           connectRoom,
		"describeroom": describeRoom,
		"dr":           describeRoom,
		"roomid":       RoomId,
		"createitem":   createItem,
		"ci":           createItem,
		"createnpc":    createNpc,
		"cn":           createNpc,
		"describeitem": describeItem,
		"di":           describeItem,
		"describenpc":  describeNpc,
		"dn":           describeNpc,
		"animate":      animate,
		"an":           animate,
		"help":         help,
		"?":            help,
		// directions
		"south":     walkSouth,
		"s":         walkSouth,
		"north":     walkNorth,
		"n":         walkNorth,
		"east":      walkEast,
		"e":         walkEast,
		"west":      walkWest,
		"w":         walkWest,
		"northeast": walkNortheast,
		"ne":        walkNortheast,
		"northwest": walkNorthwest,
		"nw":        walkNorthwest,
		"southeast": walkSoutheast,
		"se":        walkSoutheast,
		"southwest": walkSouthwest,
		"sw":        walkSouthwest,
		// items
		"get":       get,
		"g":         get,
		"drop":      drop,
		"items":     items,
		"ii":        items,
		"inventory": inventory,
		"inv":       inventory,
		"i":         inventory,
		"itemshere": itemsHere,
		"ih":        itemsHere,
		// basic commands
		"look":      look,
		"l":         look,
		"quicklook": quicklook,
		"ql":        quicklook,
		"say":       say,
		"'":         say,
		"tell":      tell,
	}
}
