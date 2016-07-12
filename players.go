/*
players.go contains Player types and funcs,
along with an PlayerManager type which
provides room-related functions for ThingManager

Player implements the Thing interface.
PlayerManager is a ThingManager

*/
package main

import (
	"fmt"
	"net"
	"strconv"
	//	 "runtime/debug"
)

type PlayerItemType int32

const (
	piItem = iota
	piNpc
)

//
// player
//
// This data must be non-volatile.
// If the server closes and reopens, it must persist
//
type Player struct {
	id          identifier
	name        string
	passthesalt []byte
	pass        []byte
	connection  net.Conn
	level       uint
	health      uint
	mana        uint
	Room        identifier
	Items       map[identifier]PlayerItemType
}

/// @todo change this to write to a channel for a manager, to prevent concurrent access to the connection
func (p *Player) Write(message string) {
	if len(message) == 0 {
		fmt.Println("player.Write called with empty string '" + p.Name() + "'")
	}
	p.connection.Write([]byte("\r\n" + message + "\r\n" + p.Prompt()))
}

func (p *Player) Id() identifier {
	return p.id
}

func (p *Player) SetId(newId identifier) {
	p.id = newId
}

func (p *Player) Name() string {
	return p.name
}

func (p *Player) MaxHealth() uint {
	return 500 * p.level
}

/// @todo ? remove the changePlayer call, and require users to call it themselves ?
func (p *Player) Injure(damage uint, world *World) {
	world.players.ChangeById(p.Id(), func(player *Player) {
		if damage > player.health {
			player.health = 0
		} else {
			player.health -= damage
		}
		if player.health == 0 {
			go player.Kill(world) // This MUST be in a goroutine; the playerManager CANNOT be called in its own routine
		}
	})
}

func (p *Player) InjureAlreadyGot(damage uint, world *World, playerSet SetterMsg) error {
	player, ok := playerSet.it.(*Player)
	if !ok {
		return fmt.Errorf("InjureAlreadyGot error: setterMsg was not a *Player")
	}
	if damage > player.health {
		player.health = 0
	} else {
		player.health -= damage
	}
	if player.health == 0 {
		go player.Kill(world) // This MUST be in a goroutine; the playerManager CANNOT be called in its own routine
	}
	return nil
}

/// This should rarely be called, e.g. with Instakills
/// Ordinarily, Injure should be called, which will call this if necessary
func (p *Player) Kill(world *World) {
	p.Write(Red + "You have died." + Reset)
	room, ok := world.rooms.GetById(p.Room)
	if !ok {
		fmt.Println("kill called with player with invalid room '" + p.Name() + "' " + p.Room.String())
		return
	}
	room.Write(p.Name()+" has died.", *world.players, p.Name())
}

func (p *Player) MaxMana() uint {
	return 300 * p.level
}

// @todo change this to refresh the player from the playerManager
//
func (p *Player) Prompt() string {
	return Green + strconv.FormatUint(uint64(p.health), 10) + "h, " +
		Blue + strconv.FormatUint(uint64(p.mana), 10) + "m" +
		Reset + "-"
}

type PlayerManager ThingManager

/// @todo remove this, after changing things which call it to store Accessors rather than IDs
/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m PlayerManager) GetById(id identifier) (*Player, bool) {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("PlayerManager.GetById error: ThingGetter nil " + id.String())
		return &Player{}, false
	}
	thing, ok := <-accessor.ThingGetter
	if !ok {
		fmt.Println("PlayerManager.GetById error: player ThingGetter closed " + id.String())
		return &Player{}, false
	}
	player, ok := thing.(*Player)
	if !ok {
		fmt.Println("PlayerManager.GetById error: player accessor returned non-player " + id.String())
		return &Player{}, false
	}
	return player, ok
}

func (m PlayerManager) GetByName(name string) (*Player, bool) {
	accessor := ThingManager(m).GetThingAccessorByName(name)
	if accessor.ThingGetter == nil {
		fmt.Println("PlayerManager.GetByName error: ThingGetter nil " + name)
		return &Player{}, false
	}
	thing, ok := <-accessor.ThingGetter
	if !ok {
		fmt.Println("PlayerManager.GetByName error: player ThingGetter closed " + name)
		return &Player{}, false
	}
	player, ok := thing.(*Player)
	if !ok {
		fmt.Println("PlayerManager.GetByName error: player accessor returned non-player " + name)
		return &Player{}, false
	}
	return player, ok
}

/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m PlayerManager) ChangeById(id identifier, modify func(p *Player)) bool {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("PlayerManager.ChangeById error: ThingGetter nil " + id.String())
		//		debug.PrintStack()
		return false
	}
	setMsg, ok := <-accessor.ThingSetter
	if !ok {
		fmt.Println("PlayerManager.ChangeById error: player ThingGetter closed " + id.String())
		return false
	}
	setMsg.chainTime <- NotChaining
	player, ok := setMsg.it.(*Player)
	if !ok {
		fmt.Println("PlayerManager.ChangeById error: player accessor returned non-player " + id.String())
		return false
	}
	modify(player)
	setMsg.set <- player
	return true
}

/// @todo move this to MetaManager ? Player ?
/// @todo change players and rooms to use Accessors rather than IDs; then this won't need the world.
func (m PlayerManager) Move(player identifier, direction Direction, world *World) bool {
	chainTime := <-NextChainTime
	playerAccessor := ThingManager(m).GetThingAccessor(player)
	for {
		sets := make([]SetterMsg, 0, 3)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: room chan closed " + player.String())
			return false
		} else if resetChain {
			continue
		}
		sets = append(sets, playerSet)
		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(playerSet.it.(*Player).Room)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: room chan closed " + player.String())
			ReleaseThings(sets)
			return false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)
		newRoomId, ok := roomSet.it.(*Room).Exits[direction]
		if !ok {
			playerSet.it.(*Player).Write("There is no exit in that direction.") // @todo ? should callers write this?
			ReleaseThings(sets)
			return false
		}
		newRoomAccessor := ThingManager(*world.rooms).GetThingAccessor(newRoomId)
		newRoomSet, ok, resetChain := newRoomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("PlayerManager.Move error: new room chan closed " + player.String())
			ReleaseThings(sets)
			return false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, newRoomSet)
		player := playerSet.it.(*Player)
		room := roomSet.it.(*Room)
		newRoom := newRoomSet.it.(*Room)
		player.Room = newRoomId
		delete(room.Players, player.Id())
		newRoom.Players[player.Id()] = true
		player.Write("You move out to the " + direction.String() + ".")
		room.Write(ToProper(player.Name())+" moves out to the "+direction.String()+".", *world.players, player.Name())
		newRoom.Write(ToProper(player.Name())+" enters from the "+direction.reverse().String()+".", *world.players, player.Name())
		player.Write(newRoom.PrintBrief(world, player.Name()))
		ReleaseThings(sets)
		break
	}
	return true
}
