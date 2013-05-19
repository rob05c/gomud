package main

import (
	"fmt"
	"github.com/mattn/go-v8"
	"math/rand"
	"strconv"
	"time"
)

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
	}
	itemLocation, itemLocationType, exists := world.itemLocations.itemLocation(itemIdentifier(itemId))
	if !exists {
		fmt.Println("jsRandomMove got nonlocated item '" + itemId.String() + "'")
	}
	if itemLocationType != ilRoom {
		fmt.Println("jsRandomMove got nonroomed item '" + itemId.String() + "'")
	}
	currentRoom, exists := world.rooms.getRoom(roomIdentifier(itemLocation))
	if !exists {
		fmt.Println("jsRandomMove getroom failed for '" + itemId.String() + "'")
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

	return world.script
}

//(function() {mud_moveRandom(self);window.setInterval(arguments.callee, 5000);})()
