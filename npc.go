package main

// I'm trying something different for NPCs.
// Rather than managing their data with Managers like other data currently is, 
// Each NPC will manage itself, with its own goroutine

type npcIdentifier identifier

func (i npcIdentifier) String() string {
	return identifier(i).String()
}

type npc struct {
	id    itemIdentifier
	name  string
	brief string
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
