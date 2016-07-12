package main

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"strconv"
	"time"
)

// funcs returns a map of lua.Functions to be registered, creating closures with the world and npc.
func funcs(world *World, npcId identifier) map[string]lua.Function {
	return map[string]lua.Function{
		"gomud_println":     luaPrintln,
		"gomud_roomPlayers": luaGetRoomPlayersFunc(world, npcId),
		"gomud_reval":       luaRevalFunc(world, npcId),
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

// luaRevalFunc takes a wait time in milliseconds from the script,
// and re-executes the npc.Animate after the wait time.
// Scripts should call this at the end of their animation right before returning.
// TODO add a min and max cap, possibly allowing admins to create NPCs under the min cap.
// TODO find a way to prevent users calling reval() and not returning (because that would have 2 threads accessing lua.State, and it's not threadsafe)
// example test lua:
// gomud_println("bill is here. You should feel honoured."); gomud_reval(1000)
func luaRevalFunc(world *World, npcId identifier) lua.Function {
	return func(l *lua.State) int {
		n := l.Top() // Number of arguments.
		if n != 1 {
			l.PushString("incorrect number of arguments: expected 1 got " + strconv.Itoa(n))
			l.Error() // panics
		}

		waitMs, ok := l.ToInteger(1)
		if !ok {
			l.PushString("incorrect argument type: expected integer")
			l.Error() // panics
		}

		go func() {
			time.Sleep(time.Duration(waitMs) * time.Millisecond)
			self, ok := world.npcs.GetById(npcId)
			if !ok {
				fmt.Println("npcReval npc nonexistent '" + npcId.String() + "'")
				return
			}
			self.Animate(world)
		}()

		return 0
	}
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

// luaGetRoomPlayers returns/pushes an array/table of the players in the room with the given npc
// example test lua:
// players = gomud_roomPlayers(); for i,player in pairs(players) do gomud_println(player) end
func luaGetRoomPlayersFunc(world *World, npcId identifier) lua.Function {
	return func(l *lua.State) int {
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
}
