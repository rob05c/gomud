package main

const defaultPort = 9241

const commandRejectMessage = "I don't understand."

func initialize() {
	initCommands()
	rooms[0] = &room {
		id: 0,
		name: "The Beginning",
		description: "Everything has a beginning. This is only one of many beginnings you will soon find as I continue typing in order to create a wall of text to test this. It's a very long sentence that precedes this slightly shorter one. Blarglblargl.",
		exits: make(map[Direction] *room),
	}
	go playerManager()
	initialPlayer := player_state{name: "rob", roomId: 0}
	createPlayer(initialPlayer)
}

func main() {
	initialize()
	listen()
}
