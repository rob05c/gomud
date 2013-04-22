package main

import (
	"fmt"
	"net"

//	"strconv"
)

type itemLocationType int32

const (
	ilRoom = iota
	ilPlayer
	ilNpc
)

type itemLocationManager struct {
	// should get called when an item is created
	itemLocationAddChan chan struct {
		itemId       itemIdentifier
		location     identifier
		locationType itemLocationType
	}

	// should be called when an item is destroyed
	itemLocationRemoveChan chan itemIdentifier

	getItemLocationChan chan struct {
		itemId   itemIdentifier
		response chan struct {
			location     identifier
			locationType itemLocationType
			exists       bool
		}
	}
	getLocationItemsChan chan struct {
		location     identifier
		locationType itemLocationType
		response     chan []itemIdentifier
	}

	itemMoveChan chan struct {
		itemId          itemIdentifier
		newLocation     identifier
		newLocationType itemLocationType
		postFunc        func(bool)
	}

	itemMoveCheckedChan chan struct {
		itemId          itemIdentifier
		oldLocation     identifier
		oldLocationType itemLocationType
		newLocation     identifier
		newLocationType itemLocationType
		postFunc        func(bool)
	}
}

func (m itemLocationManager) itemLocation(id itemIdentifier) (identifier, itemLocationType, bool) {
	responseChan := make(chan struct {
		location     identifier
		locationType itemLocationType
		exists       bool
	})
	m.getItemLocationChan <- struct {
		itemId   itemIdentifier
		response chan struct {
			location     identifier
			locationType itemLocationType
			exists       bool
		}
	}{id, responseChan}
	r := <-responseChan
	return r.location, r.locationType, r.exists
}

func (m itemLocationManager) locationItems(location identifier, locationType itemLocationType) []itemIdentifier {
	responseChan := make(chan []itemIdentifier)
	m.getLocationItemsChan <- struct {
		location     identifier
		locationType itemLocationType
		response     chan []itemIdentifier
	}{location, locationType, responseChan}
	return <-responseChan
}

func (m itemLocationManager) addItem(id itemIdentifier, location identifier, locationType itemLocationType) {
	m.itemLocationAddChan <- struct {
		itemId       itemIdentifier
		location     identifier
		locationType itemLocationType
	}{id, location, locationType}
}

func (m itemLocationManager) removeItem(id itemIdentifier) {
	m.itemLocationRemoveChan <- id
}

func (m itemLocationManager) teleportItem(c net.Conn, id itemIdentifier, location identifier, locationType itemLocationType, postFunc func(bool)) {
	m.itemMoveChan <- struct {
		itemId          itemIdentifier
		newLocation     identifier
		newLocationType itemLocationType
		postFunc        func(bool)
	}{id, location, locationType, func(success bool) {
		if !success {
			// @todo tell the user where. E.g. "it is not in your inventory," "it is not on the ground"
			c.Write([]byte("The item to move is not here."))
			return
		}
		c.Write([]byte("Item successfully moved."))
	}}
}

func (m itemLocationManager) moveItem(c net.Conn,
	id itemIdentifier,
	oldLocation identifier,
	oldLocationType itemLocationType,
	location identifier,
	locationType itemLocationType,
	postFunc func(bool)) {
	m.itemMoveCheckedChan <- struct {
		itemId          itemIdentifier
		oldLocation     identifier
		oldLocationType itemLocationType
		newLocation     identifier
		newLocationType itemLocationType
		postFunc        func(bool)
	}{id, oldLocation, oldLocationType, location, locationType, postFunc}
}

func newItemLocationManager() *itemLocationManager {
	itemLocationManager := &itemLocationManager{
		itemLocationAddChan: make(chan struct {
			itemId       itemIdentifier
			location     identifier
			locationType itemLocationType
		}),
		itemLocationRemoveChan: make(chan itemIdentifier),
		getItemLocationChan: make(chan struct {
			itemId   itemIdentifier
			response chan struct {
				location     identifier
				locationType itemLocationType
				exists       bool
			}
		}),
		getLocationItemsChan: make(chan struct {
			location     identifier
			locationType itemLocationType
			response     chan []itemIdentifier
		}),
		itemMoveChan: make(chan struct {
			itemId          itemIdentifier
			newLocation     identifier
			newLocationType itemLocationType
			postFunc        func(bool)
		}),
		itemMoveCheckedChan: make(chan struct {
			itemId          itemIdentifier
			oldLocation     identifier
			oldLocationType itemLocationType
			newLocation     identifier
			newLocationType itemLocationType
			postFunc        func(bool)
		})}
	go manageItemLocations(itemLocationManager)
	return itemLocationManager
}

func manageItemLocations(manager *itemLocationManager) {
	itemLocations := map[itemIdentifier]struct {
		location     identifier
		locationType itemLocationType
	}{}
	locationItems := map[struct {
		location     identifier
		locationType itemLocationType
	}]map[itemIdentifier]bool{}

	make_location := func(location identifier, locationType itemLocationType) struct {
		location     identifier
		locationType itemLocationType
	} {
		return struct {
			location     identifier
			locationType itemLocationType
		}{location, locationType}
	}

	checkLocationMap := func(location identifier, locationType itemLocationType) {
		if locationItems[make_location(location, locationType)] == nil {
			locationItems[make_location(location, locationType)] = map[itemIdentifier]bool{}
		}
	}

	moveItem := func(id itemIdentifier, location identifier, locationType itemLocationType) {
		oldLocation := itemLocations[id]
		itemLocations[id] = make_location(location, locationType)
		delete(locationItems[oldLocation], id)
		checkLocationMap(location, locationType)
		locationItems[make_location(location, locationType)][id] = true
	}

	for {
		select {
		case a := <-manager.itemLocationAddChan:
			if _, exists := itemLocations[a.itemId]; exists {
				fmt.Println("itemLocationManager error: add called for existing item " + a.itemId.String())
				continue
			}
			location := make_location(a.location, a.locationType)
			itemLocations[a.itemId] = location
			checkLocationMap(a.location, a.locationType)
			locationItems[location][a.itemId] = true
			fmt.Println("added item successfully")
		case r := <-manager.itemLocationRemoveChan:
			if _, exists := itemLocations[r]; !exists {
				fmt.Println("itemLocationManager error: remove called for nonexistent item " + r.String())
				continue
			}
			delete(locationItems[itemLocations[r]], r)
			delete(itemLocations, r)
		case l := <-manager.getItemLocationChan:
			location, exists := itemLocations[l.itemId]
			l.response <- struct {
				location     identifier
				locationType itemLocationType
				exists       bool
			}{location.location, location.locationType, exists}
		case i := <-manager.getLocationItemsChan:
			itemList := make([]itemIdentifier, 0)
			checkLocationMap(i.location, i.locationType)
			for key, _ := range locationItems[make_location(i.location, i.locationType)] {
				itemList = append(itemList, key)
			}
			i.response <- itemList
		case c := <-manager.itemMoveCheckedChan:
			if itemLocations[c.itemId] != make_location(c.oldLocation, c.oldLocationType) {
				go c.postFunc(false)
				continue
			}
			moveItem(c.itemId, c.newLocation, c.newLocationType)
			go c.postFunc(true)
		case m := <-manager.itemMoveChan:
			moveItem(m.itemId, m.newLocation, m.newLocationType)
			go m.postFunc(true)
		}
	}
}
