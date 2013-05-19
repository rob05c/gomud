package main

import (
	"fmt"
)

// I'm trying something different for NPCs.
// Rather than managing their data with Managers like other data currently is, 
// Each NPC will manage itself, with its own goroutine

type npcIdentifier identifier

func (i npcIdentifier) String() string {
	return identifier(i).String()
}

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
type npc struct {
	id       itemIdentifier
	name     string
	brief    string
	sleeping bool
	dna      string
}

func (n npc) Id() itemIdentifier {
	return n.id
}

func (n npc) Name() string {
	return n.name
}

func (n npc) Brief() string {
	return n.brief
}

func (n npc) Sleeping() bool {
	return n.sleeping
}

func (n npc) Dna() string {
	return n.dna
}

func (n npc) selfWrappedDna() string {
	return "(function(self) {" + n.dna + "})(" + n.id.String() + ")"
}

func (n npc) Animate(world *metaManager) {
	if n.dna == "" {
		fmt.Println("Could not animate lifeless " + n.Name())
		return
	}
	n.sleeping = false
	world.script.Eval(n.selfWrappedDna())
}
