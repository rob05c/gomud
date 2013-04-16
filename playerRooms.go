package main

import (
	"fmt"
	"strconv"
	"net"
)

// voluntary movement, e.g. 'walk s', should always go thru.
// forced movement, e.g. 'shove jim s', should only go thru if jim is in the expected location, e.g. the shover's.

// @todo change this to report to the player when movement is unsuccessful, and why
//       after a playerConnectionManager exists for us to query

/// this should only be called when a player logs in
var playerRoomAdd = make(chan struct{player string; roomId roomIdentifier})
/// this should only be called when a player logs out
var playerRoomRemove = make(chan string)

var getPlayerRoom = make(chan struct{player string; response chan struct{roomIdentifier; bool}})
var getRoomPlayers = make(chan struct{roomId roomIdentifier; response chan map[string] bool})

var playerMove = make(chan struct {player string; direction Direction; postFunc func(bool)})
var playerTeleport = make(chan struct {player string; roomId roomIdentifier; postFunc func(bool)})
var playerForceMove = make(chan struct {player string; roomId roomIdentifier; direction Direction; postFunc func(bool)})
var playerForceTeleport = make(chan struct {player string; currentRoom roomIdentifier; newRoom roomIdentifier; postFunc func(bool)})

func movePlayer(c net.Conn, player string, direction Direction, roomMan *roomManager) {
	playerMove<- struct{player string; direction Direction; postFunc func(bool)} {player, direction, func (success bool) {
			if !success {
				// @todo tell the user why (no exit, blocked, etc.
				c.Write([]byte("You can't go there.\n")) 
				return
			}
			go look(c, player, roomMan)
		}}
}

func playerRoom(player string) (roomIdentifier, bool) {
	responseChan := make(chan struct{roomIdentifier; bool})
	getPlayerRoom<- struct{player string; response chan struct{roomIdentifier; bool}}{player, responseChan}
	response := <-responseChan
	return response.roomIdentifier, response.bool
}

func roomPlayers(r roomIdentifier) map[string] bool {
	responseChan := make(chan map[string] bool)
	getRoomPlayers<- struct{roomId roomIdentifier; response chan map[string] bool}{r, responseChan}
	return <-responseChan
}

func playerRoomManager(roomMan *roomManager) {
	playerRooms := map[string] roomIdentifier {}
	roomPlayers := map[roomIdentifier] map[string] bool {}

	checkRoomMap := func(roomId roomIdentifier) { 
		if roomPlayers[roomId] == nil {
			roomPlayers[roomId] = map[string] bool {} // @todo figure out how to avoid this check
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
		case l := <- getPlayerRoom:
			roomId, exists := playerRooms[l.player]
			l.response <- struct{roomIdentifier; bool}{roomId, exists}
		case o := <- getRoomPlayers:
			var playersCopy map[string] bool
			checkRoomMap(o.roomId)
			for key, value := range roomPlayers[o.roomId] {
				playersCopy[key] = value
			}
			o.response <- playersCopy
		case a := <- playerRoomAdd:
			if _, exists := playerRooms[a.player]; exists {
				fmt.Println("playerRoomManager error: add called for existing player " + a.player)
				continue
			}
			playerRooms[a.player] = a.roomId
			checkRoomMap(a.roomId)
			roomPlayers[a.roomId][a.player] = true
		case r := <- playerRoomRemove:
			if _, exists := playerRooms[r]; !exists {
				fmt.Println("playerRoomManager error: remove called for nonexisting player " + r)
				continue
			}
			delete(roomPlayers[playerRooms[r]], r)
			delete(playerRooms, r)
		case m := <- playerMove:
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
		case t := <- playerTeleport:
			movePlayer(t.player, t.roomId)
			go t.postFunc(true)
		case f := <- playerForceMove:
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
		case g := <- playerForceTeleport:
			if playerRooms[g.player] != g.currentRoom {
				go g.postFunc(false)
				continue
			}
			movePlayer(g.player, g.newRoom)
			go g.postFunc(true)
		}
	}
}
