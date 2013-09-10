/*
players.go contains Room types and funcs,
along with an RoomManager type which
provides room-related functions for ThingManager

Room implements the Thing interface.
RoomManager is a ThingManager

*/
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

func (r Room) Print(world *World, playerName string) string {
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
func (r Room) PrintBrief(world *World, playerName string) string {
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

func (r Room) printItems(world *World) string {
	var buffer bytes.Buffer
	buffer.WriteString(Blue)
	buffer.WriteString("You see ")
	var items []string
	for id, itemType := range r.Items {
		switch itemType {
		case piItem:
			it, exists := world.items.GetById(id)
			if !exists {
				continue
			}
			items = append(items, it.Brief())
		case piNpc:
			it, exists := world.npcs.GetById(id)
			if !exists {
				continue
			}
			items = append(items, it.Brief)
		default:
			fmt.Println("room.printItems got invalid item type  '" + id.String() + "'")
			continue
		}
	}
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		buffer.WriteString(items[0])
		buffer.WriteString(" here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	if len(items) == 2 {
		buffer.WriteString(items[0])
		buffer.WriteString(" and ")
		buffer.WriteString(items[1])
		buffer.WriteString(" here.\r\n")
		buffer.WriteString(Reset)
		return buffer.String()
	}
	lastItem := items[len(items)-1]
	items = items[0 : len(items)-1]
	for _, item := range items {
		buffer.WriteString(item)
		buffer.WriteString(", ")
	}

	buffer.WriteString("and ")
	buffer.WriteString(lastItem)
	buffer.WriteString(" here.\r\n")
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r Room) printPlayers(world *World, currentPlayerName string) string {
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
