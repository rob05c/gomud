package main

import (
	"fmt"
	"github.com/mattn/go-v8"
	"math/rand"
	"strconv"
	"time"
)

/// be aware this will need to include damage type in the future. 
/// For example, the npc might do fire damage to a player wearing a fire resistance ring.
/// @todo add custom attack message
func jsAttackPlayer(itemId identifier, playerName string, baseDamage uint, world *metaManager) {
	locationId, locationType, exists := world.itemLocations.itemLocation(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsAttackPlayer got nonexistent room for " + itemId.String())
		return
	}
	if locationType != ilRoom {
		fmt.Println("jsAttackPlayer npc not in room " + itemId.String())
		return
	}
	it, exists := world.items.getItem(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsAttackPlayer item nonexistent '" + itemId.String() + "'")
		return
	}
	player, exists := world.players.getPlayer(playerName)
	if !exists {
		fmt.Println("jsAttackPlayer: player doesn't exist " + playerName)
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomIdentifier(locationId))
	if !exists {
		fmt.Println("jsAttackPlayer getroom failed for '" + itemId.String() + "'")
		return
	}
	itemSuccess := world.itemLocations.lockLocation(itemIdentifier(itemId), locationId, locationType, func() {
		/// @todo change this to use the itemLocationManager's playerLocationManager
		/// While _this_ is safe, in general, execution functions shouldn't call external managers. Else, deadlock.
		playerSuccess := world.playerLocations.lockLocation(playerName, roomIdentifier(locationId), func() {
			// things like armor, damage type resistance will need to be factored in here
			/// @todo change this to use the playerLocationManager's playerManager
			/// @todo fix this race condition. We have to launch goroutines because we have all the locks,
			///       but room.Write() requires the playerLocation manager. By running it in a goroutine, it 
			///       won't execute until the locks are released.
			///       Thus, we asserted the item and player were in the same room,
			///       But we failed to assert they remain in the same room for the attack message. Enh.
			go func() {
				player.Injure(baseDamage, world)
				player.Write(it.Brief() + " attacks you viciously.")
				currentRoom.Write(it.Brief()+" attacks "+playerName+" viciously.", world.playerLocations, playerName)
			}()
		})
		if !playerSuccess {
			fmt.Println("jsAttackPlayer npc room lock failed " + itemId.String())
		}
	})
	if !itemSuccess {
		fmt.Println("jsAttackPlayer npc room lock failed " + itemId.String())
		return
	}
}

func jsGetRoomPlayers(itemId identifier, world *metaManager) interface{} {
	itemLocation, itemLocationType, exists := world.itemLocations.itemLocation(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsGetRoomPlayers got nonlocated item '" + itemId.String() + "'")
		return nil
	}
	if itemLocationType != ilRoom {
		fmt.Println("jsGetRoomPlayers got nonroomed item '" + itemId.String() + "'")
		return nil
	}

	/*
		players := world.playerLocations.roomPlayers(roomIdentifier(itemLocation))
		jsPlayers := make([]struct {Id int; Name string}, len(players))
		for value := range players {
			jsPlayers = append(jsPlayers, struct {Id int; Name string}{Id: int(value.Id()), Name: value.Name()})
		}
	*/
	return world.playerLocations.roomPlayers(roomIdentifier(itemLocation))
}

func jsReval(itemId identifier, waitMs int, world *metaManager) {
	go func() {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
		it, exists := world.items.getItem(itemIdentifier(itemId))
		if !exists {
			fmt.Println("jsReval item nonexistent '" + itemId.String() + "'")
			return
		}
		npc, ok := it.(npc)
		if !ok {
			fmt.Println("jsReval item not npc '" + itemId.String() + "'")
			return
		}
		npc.Animate(world)
	}()
}

func jsRandomMove(itemId identifier, world *metaManager) {
	it, exists := world.items.getItem(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsRandomMove got nonexistent item '" + itemId.String() + "'")
		return
	}
	itemLocation, itemLocationType, exists := world.itemLocations.itemLocation(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsRandomMove got nonlocated item '" + itemId.String() + "'")
		return
	}
	if itemLocationType != ilRoom {
		fmt.Println("jsRandomMove got nonroomed item '" + itemId.String() + "'")
		return
	}
	currentRoom, exists := world.rooms.getRoom(roomIdentifier(itemLocation))
	if !exists {
		fmt.Println("jsRandomMove getroom failed for '" + itemId.String() + "'")
		return
	}
	if len(currentRoom.exits) == 0 {
		return
	}
	var currentRoomDirections []Direction
	for k := range currentRoom.exits {
		currentRoomDirections = append(currentRoomDirections, k)
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomDirectionIndex := rand.Int() % len(currentRoomDirections)
	randomDirection := currentRoomDirections[randomDirectionIndex]
	newRoomId := currentRoom.exits[randomDirection]
	newRoom, exists := world.rooms.getRoom(newRoomId)
	if !exists {
		fmt.Println("jsRandomMove getroom failed for new room '" + itemId.String() + "'")
		return
	}
	world.itemLocations.moveItem(it.Id(), identifier(currentRoom.id), ilRoom, identifier(newRoomId), ilRoom, func(success bool) {
		if success {
			newRoom.Write(it.Brief()+" enters from the "+randomDirection.reverse().String(), world.playerLocations, "") //@todo add item-specific message
			currentRoom.Write(it.Brief()+" moves out to the "+randomDirection.String(), world.playerLocations, "")      //@todo add item-specific message
		}
	})
}

func jsGetPlayer(world *metaManager, args ...interface{}) interface{} {
	if len(args) == 0 {
		fmt.Println("args zero")
		return ""
	}
	var playerName string
	var ok bool
	if playerName, ok = args[0].(string); !ok {
		fmt.Println("argzero not string")
		return ""
	}
	if len(playerName) < 3 {
		return ""
	}
	playerName = playerName[1 : len(playerName)-1]
	player, exists := world.players.getPlayer(playerName)
	if !exists {
		fmt.Println("player doesn't exist " + playerName)
		return ""
	}
	return struct {
		Id   int
		Name string
	}{Id: int(player.Id()), Name: player.Name()}
}

func jsPrintln(args ...interface{}) interface{} {
	if len(args) == 0 {
		fmt.Println("no args")
		return nil
	}
	var argString string
	var ok bool
	if argString, ok = args[0].(string); !ok {
		fmt.Printf("arg not string %T\n", args[0])
		return nil
	}
	if len(argString) < 3 {
		fmt.Println("args too short X" + argString + "x")
		return nil
	}
	argString = argString[1 : len(argString)-1]
	fmt.Println(argString)
	return nil
}

func initializeV8(world *metaManager) *v8.V8Context {
	world.script = v8.NewContext()
	world.script.AddFunc("mud_println", jsPrintln)
	world.script.AddFunc("mud_getPlayer", func(args ...interface{}) interface{} {
		if len(args) == 0 {
			return nil
		}
		return jsGetPlayer(world, args[0])
	})

	world.script.AddFunc("mud_moveRandom", func(args ...interface{}) interface{} {
		if len(args) == 0 {
			return nil
		}
		var selfIdString string
		var ok bool
		if selfIdString, ok = args[0].(string); !ok {
			fmt.Println("mud_moveRandom self was not a string")
			return nil
		}
		var selfId int
		var err error
		if selfId, err = strconv.Atoi(selfIdString); err != nil {
			fmt.Println("mud_moveRandom npcId not integral")
			return nil
		}
		jsRandomMove(identifier(selfId), world)
		return nil
	})

	world.script.AddFunc("mud_reval", func(args ...interface{}) interface{} {
		if len(args) < 2 {
			fmt.Println("args len insufficient " + strconv.Itoa(len(args)))
			return nil
		}
		var selfIdString string
		var ok bool
		if selfIdString, ok = args[0].(string); !ok {
			fmt.Println("mud_reval self was not a string")
			return nil
		}
		var selfId int
		var err error
		if selfId, err = strconv.Atoi(selfIdString); err != nil {
			fmt.Println("mud_reval npcId not integral")
			return nil
		}

		var waitMsString string
		if waitMsString, ok = args[1].(string); !ok {
			fmt.Println("mud_reval waitMs was not string")
			return nil
		}
		var waitMs int
		if waitMs, err = strconv.Atoi(waitMsString); err != nil {
			fmt.Println("mud_reval waitMs not integral")
			return nil
		}

		jsReval(identifier(selfId), waitMs, world)
		return nil
	})

	world.script.AddFunc("mud_roomPlayers", func(args ...interface{}) interface{} {
		if len(args) == 0 {
			fmt.Println("mud_roomPlayers empty args")
			return nil
		}
		var selfIdString string
		var ok bool
		if selfIdString, ok = args[0].(string); !ok {
			fmt.Println("mud_roomPlayers self was not a string")
			return nil
		}
		var selfId int
		var err error
		if selfId, err = strconv.Atoi(selfIdString); err != nil {
			fmt.Println("mud_roomPlayers npcId not integral")
			return nil
		}
		return jsGetRoomPlayers(identifier(selfId), world)
	})

	world.script.AddFunc("mud_attackPlayer", func(args ...interface{}) interface{} {
		if len(args) < 3 {
			fmt.Println("mud_attackPlayer args len insufficient " + strconv.Itoa(len(args)))
			return nil
		}
		var selfIdString string
		var ok bool
		if selfIdString, ok = args[0].(string); !ok {
			fmt.Println("mud_attackPlayer self was not a string")
			return nil
		}
		var selfId int
		var err error
		if selfId, err = strconv.Atoi(selfIdString); err != nil {
			fmt.Println("mud_attackPlayer npcId not integral")
			return nil
		}

		var playerName string
		if playerName, ok = args[1].(string); !ok {
			fmt.Println("mud_attackPlayer argzero not string")
			return nil
		}
		if len(playerName) < 3 {
			fmt.Println("mud_attackPlayer playername too short")
			return nil
		}
		playerName = playerName[1 : len(playerName)-1]

		var damageString string
		if damageString, ok = args[2].(string); !ok {
			fmt.Println("mud_attackPlayer self was not a string")
			return nil
		}
		var damage uint64
		if damage, err = strconv.ParseUint(damageString, 10, 64); err != nil {
			fmt.Println("mud_attackPlayer damage not integral")
			return nil
		}
		jsAttackPlayer(identifier(selfId), playerName, uint(damage), world)
		return nil
	})

	return world.script
}

//(function() {mud_moveRandom(self);window.setInterval(arguments.callee, 5000);})()
/*
var players := mud_roomPlayers(self);
if players.length == 0 {
	mud_reval(self, 5000)
	return
}
var player = players[1]
*/

//mud_moveRandom(self);var playerNames = mud_roomlayers();if(playerNames.length > 0)var playerName = playerNames[0];mud_attackPlayer(self, playerName, 42);mud_reval(self, 5000);

/*
mud_moveRandom(self);
var playerNames = mud_roomPlayers(self);
if(playerNames.length > 0)
mud_attackPlayer(self, playerNames[0], 42);
mud_reval(self, 5000);
*/
