package main

import (
	"fmt"
	"net"
	"strconv"
)

// voluntary movement, e.g. 'walk s', should always go thru.
// forced movement, e.g. 'shove jim s', should only go thru if jim is in the expected location, e.g. the shover's.

// @todo change this to report to the player when movement is unsuccessful, and why
//       after a playerConnectionManager exists for us to query

type playerLocationManager struct {
	/// this should only be called when a player logs in
	playerRoomAddChan chan struct {
		player string
		roomId roomIdentifier
	}
	/// this should only be called when a player logs out
	playerRoomRemoveChan chan string

	getPlayerRoomChan chan struct {
		player   string
		response chan struct {
			roomIdentifier
			bool
		}
	}
	getRoomPlayersChan chan struct {
		roomId   roomIdentifier
		response chan []string
	}

	playerMoveChan chan struct {
		player    string
		direction Direction
		postFunc  func(bool)
	}
	playerTeleportChan chan struct {
		player   string
		roomId   roomIdentifier
		postFunc func(bool)
	}
	playerForceMoveChan chan struct {
		player    string
		roomId    roomIdentifier
		direction Direction
		postFunc  func(bool)
	}
	playerForceTeleportChan chan struct {
		player      string
		currentRoom roomIdentifier
		newRoom     roomIdentifier
		postFunc    func(bool)
	}
}

func (m playerLocationManager) movePlayer(c net.Conn, player string, direction Direction, postFunc func(bool)) {
	m.playerMoveChan <- struct {
		player    string
		direction Direction
		postFunc  func(bool)
	}{player, direction, postFunc}

}

func (m playerLocationManager) playerRoom(player string) (roomIdentifier, bool) {
	responseChan := make(chan struct {
		roomIdentifier
		bool
	})
	m.getPlayerRoomChan <- struct {
		player   string
		response chan struct {
			roomIdentifier
			bool
		}
	}{player, responseChan}
	response := <-responseChan
	return response.roomIdentifier, response.bool
}

func (m playerLocationManager) roomPlayers(r roomIdentifier) []string {
	responseChan := make(chan []string)
	m.getRoomPlayersChan <- struct {
		roomId   roomIdentifier
		response chan []string
	}{r, responseChan}
	return <-responseChan
}

func (m playerLocationManager) addPlayer(playerName string, roomId roomIdentifier) {
	m.playerRoomAddChan <- struct {
		player string
		roomId roomIdentifier
	}{playerName, 0}
}

func newPlayerLocationManager(roomMan *roomManager) *playerLocationManager {
	playerLocationManager := &playerLocationManager{
		playerRoomAddChan: make(chan struct {
			player string
			roomId roomIdentifier
		}), playerRoomRemoveChan: make(chan string),
		getPlayerRoomChan: make(chan struct {
			player   string
			response chan struct {
				roomIdentifier
				bool
			}
		}),
		getRoomPlayersChan: make(chan struct {
			roomId   roomIdentifier
			response chan []string
		}),
		playerMoveChan: make(chan struct {
			player    string
			direction Direction
			postFunc  func(bool)
		}),
		playerTeleportChan: make(chan struct {
			player   string
			roomId   roomIdentifier
			postFunc func(bool)
		}),
		playerForceMoveChan: make(chan struct {
			player    string
			roomId    roomIdentifier
			direction Direction
			postFunc  func(bool)
		}),
		playerForceTeleportChan: make(chan struct {
			player      string
			currentRoom roomIdentifier
			newRoom     roomIdentifier
			postFunc    func(bool)
		})}
	go managePlayerLocations(playerLocationManager, roomMan)
	return playerLocationManager
}

func managePlayerLocations(manager *playerLocationManager, roomMan *roomManager) {
	playerRooms := map[string]roomIdentifier{}
	roomPlayers := map[roomIdentifier]map[string]bool{}

	checkRoomMap := func(roomId roomIdentifier) {
		if roomPlayers[roomId] == nil {
			roomPlayers[roomId] = map[string]bool{} // @todo figure out how to avoid this check
		}
	}

	movePlayer := func(player string, newRoom roomIdentifier) {
		oldRoom := playerRooms[player]
		playerRooms[player] = newRoom
		delete(roomPlayers[oldRoom], player)
		checkRoomMap(newRoom)
		roomPlayers[newRoom][player] = true
	}

	for {
		select {
		case l := <-manager.getPlayerRoomChan:
			roomId, exists := playerRooms[l.player]
			l.response <- struct {
				roomIdentifier
				bool
			}{roomId, exists}
		case o := <-manager.getRoomPlayersChan:
			playersCopy := []string{}
			checkRoomMap(o.roomId)
			for key, _ := range roomPlayers[o.roomId] {
				playersCopy = append(playersCopy, key)
			}
			o.response <- playersCopy
		case a := <-manager.playerRoomAddChan:
			if _, exists := playerRooms[a.player]; exists {
				fmt.Println("playerRoomManager error: add called for existing player " + a.player)
				continue
			}
			playerRooms[a.player] = a.roomId
			checkRoomMap(a.roomId)
			roomPlayers[a.roomId][a.player] = true
		case r := <-manager.playerRoomRemoveChan:
			if _, exists := playerRooms[r]; !exists {
				fmt.Println("playerRoomManager error: remove called for nonexisting player " + r)
				continue
			}
			delete(roomPlayers[playerRooms[r]], r)
			delete(playerRooms, r)
		case m := <-manager.playerMoveChan:
			oldRoomId := playerRooms[m.player]
			oldRoom, exists := roomMan.getRoom(oldRoomId)
			if !exists {
				fmt.Println("playerRoomManager error: move called for nonexistent room " + strconv.Itoa(int(oldRoomId)))
				go m.postFunc(false)
				continue
			}
			newRoomId, exists := oldRoom.exits[m.direction]
			if !exists {
				go m.postFunc(false)
				continue
			}
			movePlayer(m.player, newRoomId)
			go m.postFunc(true)
		case t := <-manager.playerTeleportChan:
			movePlayer(t.player, t.roomId)
			go t.postFunc(true)
		case f := <-manager.playerForceMoveChan:
			if playerRooms[f.player] != f.roomId {
				go f.postFunc(false)
				continue
			}
			oldRoom, exists := roomMan.getRoom(f.roomId)
			if !exists {
				go f.postFunc(false)
				continue
			}
			newRoomId, exists := oldRoom.exits[f.direction]
			if !exists {
				go f.postFunc(false)
				continue
			}
			movePlayer(f.player, newRoomId)
			go f.postFunc(true)
		case g := <-manager.playerForceTeleportChan:
			if playerRooms[g.player] != g.currentRoom {
				go g.postFunc(false)
				continue
			}
			movePlayer(g.player, g.newRoom)
			go g.postFunc(true)
		}
	}
}
