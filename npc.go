/*
npc.go contains Npc types and func,
along with an NpcManager type which
provides npc-related funcs for ThingManager

Npc implements the Thing interface.
NpcManager is a ThingManager

*/
package main

import (
	"fmt"
)

/// Animation:
/// Animate will be called every time a player enters a previously-empty room
/// The proper way to Animate, is for the dna function to continuously run while
/// The NPC is "activated," e.g. a player is in the room, it's chasing someone, etc.
/// Make the dna function sleep for a set tick time via javascript setInterval().
/// When it has nothing to do, and no players are present, the function should return
/// without further intervals, and set selfNpc.sleeping to true
/// Gomud will call Animate() again when a player enters the room.
///
/// @note room entry animation may be changed to local area entry in the future.
///
type Npc struct {
	id           identifier
	name         string
	Brief        string
	Sleeping     bool
	Dna          string
	Location     identifier
	LocationType ItemLocationType    ///< @todo ? remove this ? it isn't strictly necessary, as we can type assert to find the type
	Items        map[identifier]bool // true = npc, false = item
}

func (n *Npc) Id() identifier {
	return n.id
}

func (n *Npc) SetId(id identifier) {
	n.id = id
}

func (n *Npc) Name() string {
	return n.name
}

func (n *Npc) selfWrappedDna() string {
	return "(function(self) {" + n.Dna + "})(" + n.id.String() + ")"
}

func (n *Npc) Animate(world *World) {
	if n.Dna == "" {
		fmt.Println("Could not animate lifeless " + n.Name())
		return
	}
	n.Sleeping = false
//	world.script.Eval(n.selfWrappedDna())
}

type NpcManager ThingManager

/// @todo remove this, after changing things which call it to store Accessors rather than IDs
/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m NpcManager) GetById(id identifier) (*Npc, bool) {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("NpcManager.GetById error: ThingGetter nil " + id.String())
		return &Npc{}, false
	}
	thing, ok := <-accessor.ThingGetter
	if !ok {
		fmt.Println("NpcManager.GetById error: item ThingGetter closed " + id.String())
		return &Npc{}, false
	}
	item, ok := thing.(*Npc)
	if !ok {
		fmt.Println("NpcManager.GetById error: item accessor returned non-item " + id.String())
		return &Npc{}, false
	}
	return item, ok
}

/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m NpcManager) ChangeById(id identifier, modify func(p *Npc)) bool {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("NpcManager.ChangeById error: ThingGetter nil " + id.String())
		return false
	}
	setMsg, ok := <-accessor.ThingSetter
	if !ok {
		fmt.Println("NpcManager.ChangeById error: item ThingGetter closed " + id.String())
		return false
	}
	setMsg.chainTime <- NotChaining
	item, ok := setMsg.it.(*Npc)
	if !ok {
		fmt.Println("NpcManager.ChangeById error: item accessor returned non-item " + id.String())
		return false
	}
	modify(item)
	setMsg.set <- item
	return true
}
