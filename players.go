package main

import (
	"fmt"
	"net"
	"strconv"
)

//
// player
//
// This data must be non-volatile. 
// If the server closes and reopens, it must persist
//
type player_state struct {
	id          identifier
	name        string
	passthesalt []byte
	pass        []byte
	connection  net.Conn
	level       uint
	health      uint
	mana        uint
}

func (p player_state) Write(message string) {
	p.connection.Write([]byte("\n" + message + "\n" + p.Prompt()))
}

func (p player_state) Id() identifier {
	return p.id
}

func (p player_state) Name() string {
	return p.name
}

func (p player_state) MaxHealth() uint {
	return 500 * p.level
}

/// @todo ? remove the changePlayer call, and require users to call it themselves ?
func (p player_state) Injure(damage uint, world *metaManager) {
	world.players.changePlayer(p.Name(), func(player *player_state) {
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

/// This should rarely be called, e.g. with Instakills
/// Ordinarily, Injure should be called, which will call this if necessary
func (p player_state) Kill(world *metaManager) {
	p.Write(Red + "You have died." + Reset)
	roomId, exists := world.playerLocations.playerRoom(p.Name())
	if !exists {
		fmt.Println("kill called with invalid player '" + p.Name() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomId)
	if !exists {
		fmt.Println("kill called with player with invalid room '" + p.Name() + "' " + roomId.String())
		return
	}
	currentRoom.Write(p.Name()+" has died.", world.playerLocations, p.Name())
}

func (p player_state) MaxMana() uint {
	return 300 * p.level
}

// @todo change this to refresh the player from the playerManager
//
func (p player_state) Prompt() string {
	return Green + strconv.FormatUint(uint64(p.health), 10) + "h, " +
		Blue + strconv.FormatUint(uint64(p.mana), 10) + "m" +
		Reset + "-"
}

type playerManager struct {
	// users of the playerManager SHOULD NOT access these directly. 
	// rather, user the accessor member functions
	requestPlayerChan chan struct {
		key      string
		response chan struct {
			player_state
			bool
		}
	}
	requestPlayerByIdChan chan struct {
		key      identifier
		response chan struct {
			player_state
			bool
		}
	}

	playerChangeChan chan struct {
		key    string
		modify func(*player_state)
	}
	playerCreateChan chan player_state
}

/// any events besides modifying the player_state in this modify func
/// MUST be called in a goroutine. Because, if they do anything that calls the
/// playerManager, it will deadlock because it's no longer listening because you're an idiot.
///
/// @todo change this to take a post-modify func, and create a closure which calls both, to show intent
func (p playerManager) changePlayer(name string, modifier func(*player_state)) {
	p.playerChangeChan <- struct {
		key    string
		modify func(*player_state)
	}{name, modifier}
}

/// this helper function requests the player from the playerManager goroutine
func (p playerManager) getPlayer(name string) (player player_state, exists bool) {
	responseChan := make(chan struct {
		player_state
		bool
	})
	p.requestPlayerChan <- struct {
		key      string
		response chan struct {
			player_state
			bool
		}
	}{name, responseChan}
	response := <-responseChan
	return response.player_state, response.bool
}

func (p playerManager) getPlayerById(id identifier) (player player_state, exists bool) {
	responseChan := make(chan struct {
		player_state
		bool
	})
	p.requestPlayerByIdChan <- struct {
		key      identifier
		response chan struct {
			player_state
			bool
		}
	}{id, responseChan}
	response := <-responseChan
	return response.player_state, response.bool
}

func (p playerManager) createPlayer(player player_state) {
	p.playerCreateChan <- player
}

func newPlayerManager() *playerManager {
	playerManager := &playerManager{requestPlayerChan: make(chan struct {
		key      string
		response chan struct {
			player_state
			bool
		}
	}), requestPlayerByIdChan: make(chan struct {
		key      identifier
		response chan struct {
			player_state
			bool
		}
	}), playerChangeChan: make(chan struct {
		key    string
		modify func(*player_state)
	}), playerCreateChan: make(chan player_state)}
	go managePlayers(playerManager)
	return playerManager
}

func managePlayers(manager *playerManager) {
	var players = map[string]*player_state{}
	var playersById = map[identifier]*player_state{}
	for {
		select {
		case r := <-manager.requestPlayerChan:
			player, exists := players[r.key]
			var playerCopy player_state
			if exists {
				playerCopy = *player
			} else {
				playerCopy = player_state{}
			}
			r.response <- struct {
				player_state
				bool
			}{playerCopy, exists}
		case r := <-manager.requestPlayerByIdChan:
			player, exists := playersById[r.key]
			var playerCopy player_state
			if exists {
				playerCopy = *player
			} else {
				playerCopy = player_state{}
			}
			r.response <- struct {
				player_state
				bool
			}{playerCopy, exists}
		case m := <-manager.playerChangeChan:
			player, playerExists := players[m.key]
			if !playerExists {
				continue
			}
			m.modify(player)
		case n := <-manager.playerCreateChan:
			if _, exists := players[n.name]; exists {
				continue
			}
			n.id = identifier(len(players))
			players[n.name] = &n
			playersById[n.id] = &n
		}
	}
}
