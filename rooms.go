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
// Room
//
type Room struct {
	id          roomIdentifier
	name        string
	description string
	exits       map[Direction]roomIdentifier
}

func (r Room) Id() identifier {
	return identifier(r.id)
}

func (r Room) SetId(newId identifier) {
	r.id = roomIdentifier(newId)
}

func (r Room) Name() string {
	return r.name
}

func (r Room) printDirections() string {
	var buffer bytes.Buffer
	buffer.WriteString(Brown)
	if len(r.exits) == 0 {
		buffer.WriteString("You see no exits.")
	} else {
		buffer.WriteString("You see exits leading ")
		writeComma := false
		/// @todo print "and" before the last direction."
		/// @todo print "a single exit" for single exit Rooms
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

func (r Room) Print(world *metaManager, playerName string) string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\r\n")
	buffer.WriteString(Green)
	if r.description != "" {
		buffer.WriteString(r.description)
	} else {
		const noDescriptionString = "The Room seems to shimmer, as though it might fade from existence."
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
func (r Room) PrintBrief(world *metaManager, playerName string) string {
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

func (r Room) printItems(world *metaManager) string {
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

func (r Room) printPlayers(world *metaManager, currentPlayer string) string {
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
func (r *Room) NewRoom(manager *roomManager, d Direction, newName string, newDesc string) {
	newRoom := Room{
		name:        newName,
		description: newDesc,
		exits:       make(map[Direction]roomIdentifier),
	}
	newRoom.exits[d.reverse()] = r.id
	newRoomId := manager.createRoom(newRoom)
	manager.changeRoom(r.id, func(r *Room) {
		r.exits[d] = newRoomId
	})
}

func (r Room) Write(message string, playerLocations *playerLocationManager, originator string) {
	players := playerLocations.roomPlayers(r.id)
	for _, playerId := range players {
		player, exists := playerLocations.players.getPlayer(playerId)
		if !exists {
			fmt.Println("Room.Message got nonexistent player from playerLocations '" + playerId + "'")
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
			Room
			bool
		}
	}
	RoomChangeChan chan struct {
		id     roomIdentifier
		modify func(*Room)
	}
	RoomCreateChan chan struct {
		newRoom  Room
		response chan roomIdentifier
	}
}

func (m roomManager) getRoom(Roomid roomIdentifier) (newRoom Room, exists bool) {
	responseChan := make(chan struct {
		Room
		bool
	})
	m.requestRoomChan <- struct {
		id       roomIdentifier
		response chan struct {
			Room
			bool
		}
	}{Roomid, responseChan}
	response := <-responseChan
	return response.Room, response.bool
}

func (m roomManager) createRoom(r Room) roomIdentifier {
	responseChan := make(chan roomIdentifier)
	m.RoomCreateChan <- struct {
		newRoom  Room
		response chan roomIdentifier
	}{r, responseChan}
	newRoomId := <-responseChan
	return newRoomId
}

/// callers should be aware this is asynchronous - the Room is not necessarily changed immediately upon return
/// anything in modify besides modifying the Room MUST be called in a goroutine. Else, deadlock.
func (m roomManager) changeRoom(id roomIdentifier, modify func(*Room)) {
	m.RoomChangeChan <- struct {
		id     roomIdentifier
		modify func(*Room)
	}{id, modify}
}

func newRoomManager() *roomManager {
	roomManager := &roomManager{requestRoomChan: make(chan struct {
		id       roomIdentifier
		response chan struct {
			Room
			bool
		}
	}), RoomChangeChan: make(chan struct {
		id     roomIdentifier
		modify func(*Room)
	}), RoomCreateChan: make(chan struct {
		newRoom  Room
		response chan roomIdentifier
	})}
	go manageRooms(roomManager)
	return roomManager
}

func manageRooms(manager *roomManager) {
	var Rooms = map[roomIdentifier]*Room{}
	for {
		select {
		case r := <-manager.requestRoomChan:
			rRoom, exists := Rooms[r.id]
			var RoomCopy Room
			if exists {
				RoomCopy = *rRoom
			} else {
				RoomCopy = Room{id: -1}
			}
			r.response <- struct {
				Room
				bool
			}{RoomCopy, exists}
		case m := <-manager.RoomChangeChan:
			cRoom, exists := Rooms[m.id]
			if !exists {
				continue
			}
			m.modify(cRoom)
		case n := <-manager.RoomCreateChan:
			n.newRoom.id = roomIdentifier(len(Rooms))
			Rooms[n.newRoom.id] = &n.newRoom
			n.response <- n.newRoom.id
		}
	}
}
