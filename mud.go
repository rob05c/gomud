package main

const defaultPort = 9241

type identifier int32

type metaManager struct {
	*playerManager
	*roomManager
}

func initialize() metaManager {
	initCommands()
	managers := metaManager{playerManager: newPlayerManager(), roomManager: newRoomManager()}
	go playerRoomManager(managers.roomManager)

	initialRoom := room{
		id: roomIdentifier(0),
		name: "The Beginning",
		description: "Everything has a beginning. This is only one of many beginnings you will soon find as I continue typing in order to create a wall of text to test this. It's a very long sentence that precedes this slightly shorter one. Blarglblargl.",
		exits: make(map[Direction] roomIdentifier),
	}
	managers.roomManager.createRoom(initialRoom)
	return managers
}

func main() {
	managers := initialize()
	listen(managers)
}
