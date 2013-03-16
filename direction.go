package main
import (
	"strings"
)

type Direction int32
const (
	north = iota
	south
	east
	west
	northeast
	northwest
	southeast
	southwest
)

func (d Direction) String() string {
	switch d {
	case north:
		return "north"
	case south:
		return "south"
	case east:
		return "east"
	case west:
		return "west"
	case northeast:
		return "northeast"
	case northwest:
		return "northwest"
	case southeast:
		return "southeast"
	case southwest:
		return "southwest"
	}
	return "direction_error"
}

func stringToDirection(s string) Direction {
	s = strings.ToLower(s)
	switch s {
	case "n":
		fallthrough
	case "north":
		return north
	case "s":
		fallthrough
	case "south":
		return south
	case "e":
		fallthrough
	case "east":
		return east
	case "w":
		fallthrough
	case "west":
		return west
	case "ne":
		fallthrough
	case "northeast":
		return northeast
	case "nw":
		fallthrough
	case "northwest":
		return northwest
	case "se":
		fallthrough
	case "southeast":
		return southeast
	case "sw":
		fallthrough
	case "southwest":
		return southwest
	}
	return -1

}

func (d Direction) reverse() Direction {
	switch d {
	case north:
		return south
	case south:
		return north
	case east:
		return west
	case west:
		return east
	case northeast:
		return southwest
	case northwest:
		return southeast
	case southeast:
		return northwest
	case southwest:
		return northeast
	default:
		return -1
	}
	return -1
}
