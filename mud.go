package main

const defaultPort = 9241

type identifier int32

func initialize() {
	initCommands()
	go playerManager()
	go roomManager()

	initialRoom := room{
		id: roomIdentifier(0),
		name: "The Beginning",
		description: "Everything has a beginning. This is only one of many beginnings you will soon find as I continue typing in order to create a wall of text to test this. It's a very long sentence that precedes this slightly shorter one. Blarglblargl.",
		exits: make(map[Direction] roomIdentifier),
	}
	createRoom(initialRoom)
}

func main() {
	initialize()
	listen()
}
