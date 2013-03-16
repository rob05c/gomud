package main
import (
	"bytes"
)

var rooms = map[int] *room {} 
//
// room
//
type room struct {
	id int
	name string
	description string
	exits map[Direction] *room
}

func (r room) printDirections() string {
	var buffer bytes.Buffer
	buffer.WriteString(Brown)
	if len(r.exits) == 0 {
		buffer.WriteString("You see no exits.")
	} else {
		buffer.WriteString("You see exits leading ")
		writeComma := false;
		for i, _ := range r.exits {
			if writeComma {
				buffer.WriteString(", ")
			}
				buffer.WriteString(i.String())
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

func (r *room) NewRoom(d Direction, newName string, newDesc string) room {
	newRoom := room {
		id: len(rooms),
		name: newName,
		description: newDesc,
		exits: make(map[Direction] *room),
	}
	rooms[newRoom.id] = &newRoom
	r.exits[d] = &newRoom
	newRoom.exits[d.reverse()] = r
	return newRoom
}
