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
func jsAttackPlayer(self identifier, playerName string, baseDamage uint, world *metaManager) bool {
	chainTime := <-NextChainTime
	playerAccessor := ThingManager(*world.players).GetThingAccessorByName(playerName)
	for {
		sets := make([]SetterMsg, 0, 3)
		playerSet, ok, resetChain := playerAccessor.TryGet(chainTime)
		if !ok {
			return false
		} else if resetChain {
			continue
		}
		sets = append(sets, playerSet)

		npcAccessor := ThingManager(*world.npcs).GetThingAccessor(self)
		npcSet, ok, resetChain := npcAccessor.TryGet(chainTime)
		if !ok {
			ReleaseThings(sets)
			return false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, npcSet)

		if npcSet.it.(*Npc).LocationType != ilRoom {
			ReleaseThings(sets)
			return false // npcs can only attack from rooms
		}

		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(npcSet.it.(*Npc).Location)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			ReleaseThings(sets)
			return false
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)

		if playerSet.it.(*Player).Room != roomSet.it.Id() {
			ReleaseThings(sets)
			return false
		}
		playerSet.it.(*Player).Injure(baseDamage, world)
		playerSet.it.(*Player).Write(npcSet.it.(*Npc).Brief + " attacks you viciously.")
		roomSet.it.(*Room).Write(npcSet.it.(*Npc).Brief+" attacks "+playerName+" viciously.", *world.players, playerName)
		ReleaseThings(sets)
		break
	}
	return true
}

func jsGetRoomPlayers(selfId identifier, world *metaManager) interface{} {
	self, ok := world.npcs.GetById(selfId)
	if !ok {
		fmt.Println("jsGetRoomPlayers got nonlocated npc '" + selfId.String() + "'")
		return nil
	}
	if self.LocationType != ilRoom {
		fmt.Println("jsGetRoomPlayers got npc in nonroom '" + selfId.String() + "'")
		return nil
	}
	room, ok := world.rooms.GetById(self.Location)
	if self.LocationType != ilRoom {
		fmt.Println("jsGetRoomPlayers npc room not room '" + selfId.String() + "'")
		return nil
	}
	var players []string
	for id, _ := range room.Players {
		player, ok := world.players.GetById(id)
		if !ok {
			continue
		}
		players = append(players, player.Name())
	}
	return players
}

func jsReval(selfId identifier, waitMs int, world *metaManager) {
	go func() {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
		self, ok := world.npcs.GetById(selfId)
		if !ok {
			fmt.Println("jsReval npc nonexistent '" + selfId.String() + "'")
			return
		}
		self.Animate(world)
	}()
}

func jsRandomMove(selfId identifier, world *metaManager) {
	chainTime := <-NextChainTime
	selfAccessor := ThingManager(*world.npcs).GetThingAccessor(selfId)
	for {
		sets := make([]SetterMsg, 0, 3)
		selfSet, ok, resetChain := selfAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("jsRandomMove error getting self '" + selfId.String() + "'")
			return
		} else if resetChain {
			continue
		}
		sets = append(sets, selfSet)

		if selfSet.it.(*Npc).LocationType != ilRoom {
			fmt.Println("jsRandomMove npc in nonroom  '" + selfId.String() + "'")
			ReleaseThings(sets)
			return
		}

		roomAccessor := ThingManager(*world.rooms).GetThingAccessor(selfSet.it.(*Npc).Location)
		roomSet, ok, resetChain := roomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("jsRandomMove npc room not ok '" + selfId.String() + "'")
			ReleaseThings(sets)
			return
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, roomSet)
		room := roomSet.it.(*Room)
		if len(room.Exits) == 0 {
			ReleaseThings(sets)
			return
		}
		var roomDirections []Direction
		for k := range room.Exits {
			roomDirections = append(roomDirections, k)
		}
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomDirectionIndex := rand.Int() % len(roomDirections)
		randomDirection := roomDirections[randomDirectionIndex]
		newRoomId := room.Exits[randomDirection]

		newRoomAccessor := ThingManager(*world.rooms).GetThingAccessor(newRoomId)
		newRoomSet, ok, resetChain := newRoomAccessor.TryGet(chainTime)
		if !ok {
			fmt.Println("jsRandomMove npc newRoom not ok '" + selfId.String() + "'")
			ReleaseThings(sets)
			return
		} else if resetChain {
			ReleaseThings(sets)
			continue
		}
		sets = append(sets, newRoomSet)
		self := selfSet.it.(*Npc)
		self.Location = newRoomId
		selfSet.it = self
		delete(roomSet.it.(*Room).Items, selfId)
		newRoomSet.it.(*Room).Items[selfId] = piNpc
		newRoomSet.it.(*Room).Write(self.Brief+" enters from the "+randomDirection.reverse().String(), *world.players, "") //@todo add item-specific message
		roomSet.it.(*Room).Write(self.Brief+" moves out to the "+randomDirection.String(), *world.players, "")             //@todo add item-specific message
		ReleaseThings(sets)
		break
	}
	return
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

	player, ok := world.players.GetByName(playerName)
	if !ok {
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
