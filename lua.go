package main

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"math/rand"
	"strconv"
	"time"
)

// funcs returns a map of lua.Functions to be registered, creating closures with the world and npc.
func funcs(world *World, npcId identifier) map[string]lua.Function {
	return map[string]lua.Function{
		"gomud_println":     luaPrintln,
		"gomud_roomPlayers": luaGetRoomPlayersFunc(world, npcId),
		"gomud_reval":       luaRevalFunc(world, npcId),
		"gomud_randomMove":  luaRandomMoveFunc(world, npcId),
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

// luaRandomMoveFunc moves the given NPC selfId in a random direction
// example test lua:
// gomud_randomMove(); gomud_reval()
func luaRandomMoveFunc(world *World, selfId identifier) lua.Function {
	return func(l *lua.State) int {
		chainTime := <-NextChainTime
		selfAccessor := ThingManager(*world.npcs).GetThingAccessor(selfId)
		for {
			sets := make([]SetterMsg, 0, 3)
			selfSet, ok, resetChain := selfAccessor.TryGet(chainTime)
			if !ok {
				fmt.Println("luaRandomMove error getting self '" + selfId.String() + "'")
				l.PushString("error: self not found")
				l.Error() // panics
			} else if resetChain {
				continue
			}
			sets = append(sets, selfSet)

			if selfSet.it.(*Npc).LocationType != ilRoom {
				fmt.Println("luaRandomMove npc in nonroom  '" + selfId.String() + "'")
				ReleaseThings(sets)
				l.PushString("error: npc in nonroom")
				l.Error() // panics
			}

			roomAccessor := ThingManager(*world.rooms).GetThingAccessor(selfSet.it.(*Npc).Location)
			roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
			if !ok {
				fmt.Println("luaRandomMove npc room not ok '" + selfId.String() + "'")
				ReleaseThings(sets)
				l.PushString("error: npc room not ok '" + selfId.String() + "'")
				l.Error() // panics
			} else if resetChain {
				ReleaseThings(sets)
				continue
			}
			sets = append(sets, roomSet)
			room := roomSet.it.(*Room)
			if len(room.Exits) == 0 {
				// no exits, no way to move
				ReleaseThings(sets)
				return 0
			}
			var roomDirections []Direction
			for k := range room.Exits {
				roomDirections = append(roomDirections, k)
			}
			rand := rand.New(rand.NewSource(time.Now().UnixNano()))
			randomDirectionIndex := rand.Int() % len(roomDirections)
			randomDirection := roomDirections[randomDirectionIndex]
			newRoomId := room.Exits[randomDirection]

			newRoomAccessor := ThingManager(*world.rooms).GetThingAccessor(newRoomId)
			newRoomSet, ok, resetChain := newRoomAccessor.TryGet(chainTime)
			if !ok {
				fmt.Println("luaRandomMove npc newRoom not ok '" + selfId.String() + "'")
				ReleaseThings(sets)
				l.PushString("luaRandomMove npc newRoom not ok '" + selfId.String() + "'")
				l.Error() // panics
			} else if resetChain {
				ReleaseThings(sets)
				continue
			}
			sets = append(sets, newRoomSet)
			self := selfSet.it.(*Npc)
			self.Location = newRoomId
			selfSet.it = self
			delete(roomSet.it.(*Room).Items, selfId)
			newRoomSet.it.(*Room).Items[selfId] = piNpc
			newRoomSet.it.(*Room).Write(self.Brief+" enters from the "+randomDirection.reverse().String(), *world.players, "") //@todo add item-specific message
			roomSet.it.(*Room).Write(self.Brief+" moves out to the "+randomDirection.String(), *world.players, "")             //@todo add item-specific message
			ReleaseThings(sets)
			break
		}
		return 0
	}
}
