package main
import (
	"bytes"
)

// @todo create roomIdentifier.String()

type roomIdentifier identifier

//
// room
//
type room struct {
	id roomIdentifier
	name string
	description string
	exits map[Direction] roomIdentifier
}

func (r room) printDirections() string {
	var buffer bytes.Buffer
	buffer.WriteString(Brown)
	if len(r.exits) == 0 {
		buffer.WriteString("You see no exits.")
	} else {
		buffer.WriteString("You see exits leading ")
		writeComma := false;
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

func (r room) Print() string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\n")
	buffer.WriteString(Green)
	if r.description != "" {
		buffer.WriteString(r.description)
	} else {
		const noDescriptionString = "The room seems to shimmer, as though it might fade from existence."
		buffer.WriteString(noDescriptionString)
	}
	buffer.WriteString("\n")
	buffer.WriteString(r.printDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

// print sans description
func (r room) PrintBrief() string {
	var buffer bytes.Buffer
	buffer.WriteString(Red)
	buffer.WriteString(r.name)
	buffer.WriteString("\n")
	buffer.WriteString(r.printDirections())
	buffer.WriteString(Reset)
	return buffer.String()
}

func (r *room) NewRoom(d Direction, newName string, newDesc string) {
	newRoom := room {
		name: newName,
		description: newDesc,
		exits: make(map[Direction] roomIdentifier),
	}
	newRoom.exits[d.reverse()] = r.id
	newRoomId := createRoom(newRoom)
	roomChange<- struct {id roomIdentifier; modify func(*room)} {r.id, func(r *room){
			r.exits[d] = newRoomId
	}}
}

func getRoom(roomid roomIdentifier) (newroom room, exists bool) {
	responseChan := make(chan struct{room; bool})
	requestRoom<- struct {id roomIdentifier; response chan struct {room; bool}}{roomid, responseChan}
	response := <-responseChan
	return response.room, response.bool
}

func createRoom(r room) roomIdentifier {
	responseChan := make(chan roomIdentifier)
	roomCreate<- struct{newroom room; response chan roomIdentifier} {r, responseChan}
	newRoomId := <-responseChan
	return newRoomId
}

/// @todo pass these around, and remove from global scope
var requestRoom = make(chan struct {id roomIdentifier; response chan struct {room; bool}})
/// anything in modify besides modifying the room MUST be called in a goroutine. Else, deadlock.
var roomChange = make(chan struct {id roomIdentifier; modify func(*room)})
var roomCreate = make(chan struct {newroom room; response chan roomIdentifier})

func roomManager() {
	var rooms = map[roomIdentifier] *room {} 
	for {
		select {
		case r := <-requestRoom:
			rroom, exists := rooms[r.id]
			var roomCopy room
			if exists {
				roomCopy = *rroom
			} else {
				roomCopy = room{id: -1}
			}
			r.response<- struct {room; bool} {roomCopy, exists}
		case m := <-roomChange:
			croom, exists := rooms[m.id]
			if !exists {
				continue
			}
			m.modify(croom)
		case n := <-roomCreate:
			n.newroom.id = roomIdentifier(len(rooms))
			rooms[n.newroom.id] = &n.newroom
			n.response<- n.newroom.id
		}
	}
}