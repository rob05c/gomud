package main

import (
	"strconv"
)

type itemIdentifier identifier

func (i itemIdentifier) String() string {
	return strconv.Itoa(int(i))
	//	return identifier(i).String()
}

type item struct {
	id     itemIdentifier
	name   string
	brief  string
	long   string
	ground string
}

type itemManager struct {
	request chan struct {
		id       itemIdentifier
		response chan struct {
			item
			bool
		}
	}
	change chan struct {
		id     itemIdentifier
		modify func(*item)
	}
	create chan struct {
		newItem  item
		response chan itemIdentifier
	}
}

func (m itemManager) getItem(itemId itemIdentifier) (newItem item, exists bool) {
	responseChan := make(chan struct {
		item
		bool
	})
	m.request <- struct {
		id       itemIdentifier
		response chan struct {
			item
			bool
		}
	}{itemId, responseChan}
	response := <-responseChan
	return response.item, response.bool
}

func (m itemManager) createItem(i item) itemIdentifier {
	responseChan := make(chan itemIdentifier)
	m.create <- struct {
		newItem  item
		response chan itemIdentifier
	}{i, responseChan}
	newItemId := <-responseChan
	return newItemId
}

/// callers should be aware this is asynchronous - the room is not necessarily changed immediately upon return
/// anything in modify besides modifying the room MUST be called in a goroutine. Else, deadlock.
func (m itemManager) changeItem(id itemIdentifier, modify func(*item)) {
	m.change <- struct {
		id     itemIdentifier
		modify func(*item)
	}{id, modify}
}

func newItemManager() *itemManager {
	itemManager := &itemManager{request: make(chan struct {
		id       itemIdentifier
		response chan struct {
			item
			bool
		}
	}), change: make(chan struct {
		id     itemIdentifier
		modify func(*item)
	}), create: make(chan struct {
		newItem  item
		response chan itemIdentifier
	})}
	go manageItems(itemManager)
	return itemManager
}

func manageItems(manager *itemManager) {
	var items = map[itemIdentifier]*item{}
	for {
		select {
		case r := <-manager.request:
			i, exists := items[r.id]
			var itemCopy item
			if exists {
				itemCopy = *i
			} else {
				itemCopy = item{id: -1}
			}
			r.response <- struct {
				item
				bool
			}{itemCopy, exists}
		case h := <-manager.change:
			i, exists := items[h.id]
			if !exists {
				continue
			}
			h.modify(i)
		case c := <-manager.create:
			c.newItem.id = itemIdentifier(len(items))
			items[c.newItem.id] = &c.newItem
			c.response <- c.newItem.id
		}
	}
}
