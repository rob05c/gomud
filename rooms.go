package main

import (
	"bytes"
	"fmt"
)

type Room struct {
	id          identifier
	name        string
	Description string
	Exits       map[Direction]identifier
	Players     map[identifier]bool
	Items       map[identifier]PlayerItemType
}

func (r *Room) Id() identifier {
	return r.id
}

func (r *Room) SetId(newId identifier) {
	r.id = newId
}

func (r *Room) Name() string {
	return r.name
}

func (r Room) PrintDirections() string {
	var buffer bytes.Buffer
	buffer.WriteString(Brown)
	if len(r.Exits) == 0 {
		buffer.WriteString("You see no exits.")
	} else {
		buffer.WriteString("You see exits leading ")
		writeComma := false
		/// @todo print "and" before the last direction."
		/// @todo print "a single exit" for single exit Rooms
		for d := range r.Exits {
			if writeComma {
				buffer.WriteString(", ")
			}
			buffer.WriteString(d.String())
			writeComma = true
		}
		buffer.WriteString(".")
	}
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r Room) Print(world *metaManager, playerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\r\n")
	buffer.WriteString(Green)
	if r.Description != "" {
		buffer.WriteString(r.Description)
	} else {
		const noDescriptionString = "The Room seems to shimmer, as though it might fade from existence."
		buffer.WriteString(noDescriptionString)
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(r.printItems(world))
	buffer.WriteString(r.printPlayers(world, playerName))
	buffer.WriteString(r.PrintDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

// print sans description
func (r Room) PrintBrief(world *metaManager, playerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\r\n")
	buffer.WriteString(r.printItems(world))
	buffer.WriteString(r.printPlayers(world, playerName))
	buffer.WriteString(r.PrintDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r Room) printItems(world *metaManager) string {
	var buffer bytes.Buffer
	buffer.WriteString(Blue)
	buffer.WriteString("You see ")
	var items []identifier
	for id, _ := range r.Items {
		items = append(items, id)
	}
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		it, exists := world.items.GetById(items[0])
		if !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + items[0].String() + "'")
			return ""
		}
		buffer.WriteString(it.Brief())
		buffer.WriteString(" here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	if len(items) == 2 {
		if itemFirst, exists := world.items.GetById(items[0]); !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + items[0].String() + "'")
		} else {
			buffer.WriteString(itemFirst.Brief())
			buffer.WriteString(" and ")
		}
		if itemSecond, exists := world.items.GetById(items[1]); !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + items[1].String() + "'")
			buffer.WriteString("your shadow") // see what I did there?
		} else {
			buffer.WriteString(itemSecond.Brief())
		}
		buffer.WriteString(" here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	lastItemId := items[len(items)-1]
	items = items[0 : len(items)-1]
	for _, itemId := range items {
		it, exists := world.items.GetById(itemId)
		if !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
		}
		buffer.WriteString(it.Brief())
		buffer.WriteString(", ")
	}

	buffer.WriteString("and ")
	lastItem, exists := world.items.GetById(lastItemId)
	if !exists {
		fmt.Println("items got nonexistent item from itemLocationManager '" + lastItemId.String() + "'")
		buffer.WriteString("your shadow") // see what I did there?
	} else {
		buffer.WriteString(lastItem.Brief())
	}
	buffer.WriteString(" here.\r\n")
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r Room) printPlayers(world *metaManager, currentPlayerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Darkcyan)

	currentPlayer, ok := world.players.GetByName(currentPlayerName)
	if !ok {
		fmt.Println("Room.printPlayers failed '" + r.id.String() + "'")
		return ""
	}
	var players []string
	for playerId, _ := range r.Players {
		if playerId == currentPlayer.Id() {
			continue
		}
		player, ok := world.players.GetById(playerId)
		if !ok {
			continue
		}
		players = append(players, player.Name())

	}
	if len(players) == 0 {
		return ""
	}
	if len(players) == 1 {
		buffer.WriteString(ToProper(players[0]))
		buffer.WriteString(" is here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	if len(players) == 2 {
		buffer.WriteString(ToProper(players[0]))
		buffer.WriteString(" and ")
		buffer.WriteString(players[1])
		buffer.WriteString(" are here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	lastPlayer := players[len(players)-1]
	players = players[0 : len(players)-1]
	for _, player := range players {
		buffer.WriteString(ToProper(player))
		buffer.WriteString(", ")
	}

	buffer.WriteString("and ")
	buffer.WriteString(ToProper(lastPlayer))
	buffer.WriteString(" are here.\r\n")
	buffer.WriteString(Reset)
	return buffer.String()
}

/// @todo ? make this a member of roomManager
func (r *Room) NewRoom(manager *RoomManager, d Direction, newName string, newDesc string) {
	newRoom := Room{
		name:        newName,
		Description: newDesc,
		Exits:       make(map[Direction]identifier),
		Players:     make(map[identifier]bool),
		Items:       make(map[identifier]PlayerItemType),
	}
	newRoom.Exits[d.reverse()] = r.id
	newRoomId := ThingManager(*manager).Add(&newRoom)
	accessor := ThingManager(*manager).GetThingAccessor(r.Id())
	ok := accessor.ThingSetter.Set(func(r *Thing) {
		(*r).(*Room).Exits[d] = newRoomId
	})
	if !ok {
		fmt.Println("Room.NewRoom set failed '" + r.id.String() + "'")
	}
}

func (r Room) Write(message string, playerManager PlayerManager, originator string) {
	for pid, _ := range r.Players {
		player, exists := playerManager.GetById(pid)
		if !exists {
			fmt.Println("Room.Write got nonexistent player '" + pid.String() + "'")
			continue
		}
		if player.Name() == originator {
			continue
		}
		player.Write(message)
	}
}

type RoomManager ThingManager

/// @todo remove this, after changing things which call it to store Accessors rather than IDs
/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m RoomManager) GetById(id identifier) (*Room, bool) {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("RoomManager.GetById error: ThingGetter nil " + id.String())
		return &Room{}, false
	}
	thing, ok := <-accessor.ThingGetter
	if !ok {
		fmt.Println("RoomManager.GetById error: room ThingGetter closed " + id.String())
		return &Room{}, false
	}
	room, ok := thing.(*Room)
	if !ok {
		fmt.Println("RoomManager.GetById error: room accessor returned non-room " + id.String())
		return &Room{}, false
	}
	return room, ok
}

/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m RoomManager) ChangeById(id identifier, modify func(r *Room)) bool {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("RoomManager.ChangeById error: ThingGetter nil " + id.String())
		return false
	}
	setMsg, ok := <-accessor.ThingSetter
	if !ok {
		fmt.Println("RoomManager.ChangeById error: room ThingGetter closed " + id.String())
		return false
	}
	setMsg.chainTime <- NotChaining
	room, ok := setMsg.it.(*Room)
	if !ok {
		fmt.Println("RoomManager.ChangeById error: room accessor returned non-room " + id.String())
		return false
	}
	modify(room)
	setMsg.set <- room
	return true
}
