package main

import (
	"fmt"
)

type ItemLocationType int32

const (
	ilRoom = iota
	ilPlayer
	ilNpc
)

type Item struct {
	id           identifier
	name         string
	brief        string
	Location     identifier
	LocationType ItemLocationType ///< @todo ? remove this ? it isn't strictly necessary, as we can type assert to find the type
	Items        map[identifier]bool
}

func (i *Item) Id() identifier {
	return i.id
}

func (i *Item) SetId(newId identifier) {
	i.id = newId
}

func (i *Item) Name() string {
	return i.name
}

func (i *Item) Brief() string {
	return i.brief
}

type ItemManager ThingManager

/// @todo remove this, after changing things which call it to store Accessors rather than IDs
/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m ItemManager) GetById(id identifier) (*Item, bool) {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("ItemManager.GetById error: ThingGetter nil " + id.String())
		return &Item{}, false
	}
	thing, ok := <-accessor.ThingGetter
	if !ok {
		fmt.Println("ItemManager.GetById error: item ThingGetter closed " + id.String())
		return &Item{}, false
	}
	item, ok := thing.(*Item)
	if !ok {
		fmt.Println("ItemManager.GetById error: item accessor returned non-item " + id.String())
		return &Item{}, false
	}
	return item, ok
}

/// @todo change this to return an error object with an err string, rather than printing the err and returning bool
func (m ItemManager) ChangeById(id identifier, modify func(p *Item)) bool {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("ItemManager.ChangeById error: ThingGetter nil " + id.String())
		return false
	}
	setMsg, ok := <-accessor.ThingSetter
	if !ok {
		fmt.Println("ItemManager.ChangeById error: item ThingGetter closed " + id.String())
		return false
	}
	setMsg.chainTime <- NotChaining
	item, ok := setMsg.it.(*Item)
	if !ok {
		fmt.Println("ItemManager.ChangeById error: item accessor returned non-item " + id.String())
		return false
	}
	modify(item)
	setMsg.set <- item
	return true
}
