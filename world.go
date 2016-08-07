package main

import (
	"database/sql"
	"fmt"
)

type World struct {
	rooms   *RoomManager
	players *PlayerManager
	items   *ItemManager
	npcs    *NpcManager
	db      *sql.DB
}

type ToGet struct {
	rooms   []identifier
	players []identifier
	items   []identifier
	npcs    []identifier
}

func RoomGet(id identifier) *ToGet {
	return &ToGet{rooms: []identifier{id}, players: nil, items: nil, npcs: nil}
}
func PlayerGet(id identifier) *ToGet {
	return &ToGet{rooms: nil, players: []identifier{id}, items: nil, npcs: nil}
}
func ItemGet(id identifier) *ToGet {
	return &ToGet{rooms: nil, players: nil, items: []identifier{id}, npcs: nil}
}
func NpcGet(id identifier) *ToGet {
	return &ToGet{rooms: nil, players: nil, items: nil, npcs: []identifier{id}}
}

type Got struct {
	rooms   map[identifier]*Room
	players map[identifier]*Player
	items   map[identifier]*Item
	npcs    map[identifier]*Npc
}

func NewGot() Got {
	return Got{
		rooms:   map[identifier]*Room{},
		players: map[identifier]*Player{},
		items:   map[identifier]*Item{},
		npcs:    map[identifier]*Npc{},
	}
}

type DoFunc func(got Got) (*ToGet, error)

func removeGot(toGet *ToGet, got Got) *ToGet {
	newToGet := ToGet{}
	for _, room := range toGet.rooms {
		if _, ok := got.rooms[room]; !ok {
			newToGet.rooms = append(newToGet.rooms, room)
		}
	}
	for _, player := range toGet.players {
		if _, ok := got.players[player]; !ok {
			newToGet.players = append(newToGet.players, player)
		}
	}
	for _, item := range toGet.items {
		if _, ok := got.items[item]; !ok {
			newToGet.items = append(newToGet.items, item)
		}
	}
	for _, npc := range toGet.npcs {
		if _, ok := got.npcs[npc]; !ok {
			newToGet.npcs = append(newToGet.npcs, npc)
		}
	}
	return &newToGet
}

func getAccessors(ids []identifier, manager ThingManager) map[identifier]ThingAccessor {
	accessors := map[identifier]ThingAccessor{}
	for _, id := range ids {
		accessors[id] = manager.GetThingAccessor(id)
	}
	return accessors
}

func (w *World) Do(funcs []DoFunc) error {
	got := NewGot()
	toGet := &ToGet{}
	chainTime := <-NextChainTime

AcquireLoop:
	for {
		sets := []SetterMsg{}
		for _, f := range funcs {
			toGet = removeGot(toGet, got)
			roomAccessors := getAccessors(toGet.rooms, ThingManager(*w.rooms))
			playerAccessors := getAccessors(toGet.players, ThingManager(*w.players))
			itemAccessors := getAccessors(toGet.items, ThingManager(*w.items))
			npcAccessors := getAccessors(toGet.npcs, ThingManager(*w.npcs))

			roomSets := []SetterMsg{}
			playerSets := []SetterMsg{}
			itemSets := []SetterMsg{}
			npcSets := []SetterMsg{}

			for id, accessor := range roomAccessors {
				set, ok, resetChain := accessor.TryGet(chainTime) // TODO change TryGet ok to an error?
				if !ok {
					ReleaseThings(sets)
					return fmt.Errorf("World.Do error getting room %v", id)
				} else if resetChain {
					ReleaseThings(sets)
					continue AcquireLoop
				}
				roomSets = append(roomSets, set)
				sets = append(sets, set)
			}
			for id, accessor := range playerAccessors {
				set, ok, resetChain := accessor.TryGet(chainTime) // TODO change TryGet ok to an error?
				if !ok {
					ReleaseThings(sets)
					return fmt.Errorf("World.Do error getting player %v", id)
				} else if resetChain {
					ReleaseThings(sets)
					continue AcquireLoop
				}
				playerSets = append(playerSets, set)
				sets = append(sets, set)
			}
			for id, accessor := range itemAccessors {
				set, ok, resetChain := accessor.TryGet(chainTime) // TODO change TryGet ok to an error?
				if !ok {
					ReleaseThings(sets)
					return fmt.Errorf("World.Do error getting item %v", id)
				} else if resetChain {
					ReleaseThings(sets)
					continue AcquireLoop
				}
				itemSets = append(itemSets, set)
				sets = append(sets, set)
			}
			for id, accessor := range npcAccessors {
				set, ok, resetChain := accessor.TryGet(chainTime) // TODO change TryGet ok to an error?
				if !ok {
					ReleaseThings(sets)
					return fmt.Errorf("World.Do error getting npc %v", id)
				} else if resetChain {
					ReleaseThings(sets)
					continue AcquireLoop
				}
				npcSets = append(npcSets, set)
				sets = append(sets, set)
			}

			for _, set := range roomSets {
				if room, ok := set.it.(*Room); ok {
					got.rooms[set.it.Id()] = room
				} else {
					ReleaseThings(sets)
					return fmt.Errorf("%v not a room", set.it.Id())
				}
			}
			for _, set := range playerSets {
				if player, ok := set.it.(*Player); ok {
					got.players[set.it.Id()] = player
				} else {
					ReleaseThings(sets)
					return fmt.Errorf("%v not a player", set.it.Id())
				}
			}
			for _, set := range itemSets {
				if item, ok := set.it.(*Item); ok {
					got.items[set.it.Id()] = item
				} else {
					ReleaseThings(sets)
					return fmt.Errorf("%v not an item", set.it.Id())
				}
			}
			for _, set := range npcSets {
				if npc, ok := set.it.(*Npc); ok {
					got.npcs[set.it.Id()] = npc
				} else {
					ReleaseThings(sets)
					return fmt.Errorf("%v not an npc", set.it.Id())
				}
			}

			var err error
			toGet, err = f(got)
			if err != nil {
				ReleaseThings(sets)
				return err
			}
		}
		ReleaseThings(sets)
		return nil
	}
}
