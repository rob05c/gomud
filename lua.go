package main

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"strconv"
)

// luaNpcFunc is an NPC function, which requires the world and the NPC identifier
type luaNpcFunc func(l *lua.State, world *World, npcId identifier) int

// wrapLuaFunc takes a luaNpcFunc and closes and wraps it into a lua.Function
func wrapLuaFunc(f luaNpcFunc, world *World, npcId identifier) lua.Function {
	return func(l *lua.State) int {
		return f(l, world, npcId)
	}
}

// funcs returns a map of lua.Functions to be registered, creating closures with the world and npc.
func funcs(world *World, npcId identifier) map[string]lua.Function {
	return map[string]lua.Function{
		"gomud_println":     luaPrintln,
		"gomud_roomPlayers": wrapLuaFunc(luaGetRoomPlayers, world, npcId),
	}
}

// initLua creates a lua State with function closures with the given world and npc.
func initLua(world *World, id identifier) *lua.State {
	l := lua.NewState()
	lua.OpenLibraries(l)
	for name, f := range funcs(world, id) {
		l.Register(name, f)
	}
	return l
}

// luaPrintln prints a string to the Server console. Used for server debugging. NPC scripters should not call this, and it should not be in the user-facing help.
// example test lua:
// gomud_println("Hallo, Welt!")
func luaPrintln(l *lua.State) int {
	n := l.Top() // Number of arguments.
	if n != 1 {
		l.PushString("incorrect number of arguments: expected 1 got " + strconv.Itoa(n))
		l.Error() // panics
	}

	s, ok := l.ToString(1)
	if !ok {
		l.PushString("incorrect argument: expected string")
		l.Error() // panics
	}

	fmt.Println("Lua says: " + s)

	return 0 // Result count
}

// luaPushStrings pushes an array (table) to the stack (function return)
func luaPushStrings(l *lua.State, ss []string) {
	l.NewTable()
	for i, s := range ss {
		l.PushInteger(i)
		l.PushString(s)
		l.SetTable(-3)
	}
}

// luaGetRoomPlayers returns/pushes an array/table of the players in the room with the given npc
// example test lua:
// players = gomud_roomPlayers(); for i,player in pairs(players) do gomud_println(player) end
func luaGetRoomPlayers(l *lua.State, world *World, npcId identifier) int {
	self, ok := world.npcs.GetById(npcId)
	if !ok {
		fmt.Println("luaGetRoomPlayers got nonlocated npc '" + npcId.String() + "'")
		l.PushString("error: self not found")
		l.Error() // panics
	}

	if self.LocationType != ilRoom {
		fmt.Println("luaGetRoomPlayers got npc in nonroom '" + npcId.String() + "'")
		l.PushString("error: self in nonroom")
		l.Error() // panics
	}

	room, ok := world.rooms.GetById(self.Location)
	if self.LocationType != ilRoom {
		fmt.Println("luaGetRoomPlayers npc room not room '" + npcId.String() + "'")
		l.PushString("error: self not in room")
		l.Error() // panics
	}

	var players []string
	for id, _ := range room.Players {
		player, ok := world.players.GetById(id)
		if !ok {
			continue
		}
		players = append(players, player.Name())
	}

	luaPushStrings(l, players)
	return 1
}
