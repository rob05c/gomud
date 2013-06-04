package main

import (
	"bytes"
	"fmt"
)

// @todo create roomIdentifier.String()

type roomIdentifier identifier

func (i roomIdentifier) String() string {
	return identifier(i).String()
}

//
// room
//
type room struct {
	id          roomIdentifier
	name        string
	description string
	exits       map[Direction]roomIdentifier
}

func (r room) printDirections() string {
	var buffer bytes.Buffer
	buffer.WriteString(Brown)
	if len(r.exits) == 0 {
		buffer.WriteString("You see no exits.")
	} else {
		buffer.WriteString("You see exits leading ")
		writeComma := false
		/// @todo print "and" before the last direction."
		/// @todo print "a single exit" for single exit rooms
		for d := range r.exits {
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

func (r room) Print(world *metaManager, playerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\r\n")
	buffer.WriteString(Green)
	if r.description != "" {
		buffer.WriteString(r.description)
	} else {
		const noDescriptionString = "The room seems to shimmer, as though it might fade from existence."
		buffer.WriteString(noDescriptionString)
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(r.printItems(world))
	buffer.WriteString(r.printPlayers(world, playerName))
	buffer.WriteString(r.printDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

// print sans description
func (r room) PrintBrief(world *metaManager, playerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\r\n")
	buffer.WriteString(r.printItems(world))
	buffer.WriteString(r.printPlayers(world, playerName))
	buffer.WriteString(r.printDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r room) printItems(world *metaManager) string {
	var buffer bytes.Buffer
	buffer.WriteString(Blue)
	buffer.WriteString("You see ")
	items := world.itemLocations.locationItems(identifier(r.id), ilRoom)
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		it, exists := world.items.getItem(items[0])
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
		if itemFirst, exists := world.items.getItem(items[0]); !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + items[0].String() + "'")
		} else {
			buffer.WriteString(itemFirst.Brief())
			buffer.WriteString(" and ")
		}
		if itemSecond, exists := world.items.getItem(items[1]); !exists {
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
		it, exists := world.items.getItem(itemId)
		if !exists {
			fmt.Println("items got nonexistent item from itemLocationManager '" + itemId.String() + "'")
		}
		buffer.WriteString(it.Brief())
		buffer.WriteString(", ")
	}

	buffer.WriteString("and ")
	lastItem, exists := world.items.getItem(lastItemId)
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

func (r room) printPlayers(world *metaManager, currentPlayer string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Darkcyan)

	players := world.playerLocations.roomPlayers(r.id)
	for i, player := range players {
		if player == currentPlayer {
			players = append(players[:i], players[i+1:]...)
			break
		}
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
func (r *room) NewRoom(manager *roomManager, d Direction, newName string, newDesc string) {
	newRoom := room{
		name:        newName,
		description: newDesc,
		exits:       make(map[Direction]roomIdentifier),
	}
	newRoom.exits[d.reverse()] = r.id
	newRoomId := manager.createRoom(newRoom)
	manager.changeRoom(r.id, func(r *room) {
		r.exits[d] = newRoomId
	})
}

func (r room) Write(message string, playerLocations *playerLocationManager, originator string) {
	players := playerLocations.roomPlayers(r.id)
	for _, playerId := range players {
		player, exists := playerLocations.players.getPlayer(playerId)
		if !exists {
			fmt.Println("room.Message got nonexistent player from playerLocations '" + playerId + "'")
			continue
		}
		if playerId == originator {
			continue
		}
		player.Write(message)
	}
}

type roomManager struct {
	requestRoomChan chan struct {
		id       roomIdentifier
		response chan struct {
			room
			bool
		}
	}
	roomChangeChan chan struct {
		id     roomIdentifier
		modify func(*room)
	}
	roomCreateChan chan struct {
		newroom  room
		response chan roomIdentifier
	}
}

func (m roomManager) getRoom(roomid roomIdentifier) (newroom room, exists bool) {
	responseChan := make(chan struct {
		room
		bool
	})
	m.requestRoomChan <- struct {
		id       roomIdentifier
		response chan struct {
			room
			bool
		}
	}{roomid, responseChan}
	response := <-responseChan
	return response.room, response.bool
}

func (m roomManager) createRoom(r room) roomIdentifier {
	responseChan := make(chan roomIdentifier)
	m.roomCreateChan <- struct {
		newroom  room
		response chan roomIdentifier
	}{r, responseChan}
	newRoomId := <-responseChan
	return newRoomId
}

/// callers should be aware this is asynchronous - the room is not necessarily changed immediately upon return
/// anything in modify besides modifying the room MUST be called in a goroutine. Else, deadlock.
func (m roomManager) changeRoom(id roomIdentifier, modify func(*room)) {
	m.roomChangeChan <- struct {
		id     roomIdentifier
		modify func(*room)
	}{id, modify}
}

func newRoomManager() *roomManager {
	roomManager := &roomManager{requestRoomChan: make(chan struct {
		id       roomIdentifier
		response chan struct {
			room
			bool
		}
	}), roomChangeChan: make(chan struct {
		id     roomIdentifier
		modify func(*room)
	}), roomCreateChan: make(chan struct {
		newroom  room
		response chan roomIdentifier
	})}
	go manageRooms(roomManager)
	return roomManager
}

func manageRooms(manager *roomManager) {
	var rooms = map[roomIdentifier]*room{}
	for {
		select {
		case r := <-manager.requestRoomChan:
			rroom, exists := rooms[r.id]
			var roomCopy room
			if exists {
				roomCopy = *rroom
			} else {
				roomCopy = room{id: -1}
			}
			r.response <- struct {
				room
				bool
			}{roomCopy, exists}
		case m := <-manager.roomChangeChan:
			croom, exists := rooms[m.id]
			if !exists {
				continue
			}
			m.modify(croom)
		case n := <-manager.roomCreateChan:
			n.newroom.id = roomIdentifier(len(rooms))
			rooms[n.newroom.id] = &n.newroom
			n.response <- n.newroom.id
		}
	}
}
