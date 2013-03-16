package main

//
// player
//
// This data must be non-volatile. 
// If the server closes and reopens, it must persist
//
type player_state struct {
	name string
	roomId int
}

/// this helper function requests the player from the playerManager goroutine
func getPlayer(name string) (player player_state, exists bool) {
	responseChan := make(chan struct{player_state; bool})
	requestPlayer<- struct {key string; response chan struct {player_state; bool}}{name, responseChan}
	response := <-responseChan
	return response.player_state, response.bool
}
/// this helper func tells the PlayerManager goroutine to create a new player
/// @todo ?change playerCreate to only take the player_state and infer the name?
func createPlayer(player player_state) {
	playerCreate<- struct{name string; player player_state} {player.name, player}
}
/// @todo pass these around, and remove from global scope
var requestPlayer = make(chan struct {key string; response chan struct {player_state; bool}})
/// any events besides modifying the player_state in this modify func
/// MUST be called in a goroutine. Because, if they do anything that calls the
/// playerManager, it will deadlock because it's no longer listening because you're an idiot.
var playerChange = make(chan struct {key string; modify func(*player_state)})
var playerCreate = make(chan struct {name string; player player_state})
/// The way playerChange works: callers pass the name of the player to modify, and
/// a mutating function which takes the pointer to the real player_state in the map.
func playerManager() {
	var players =  map[string] *player_state {}
	for {
		select {
		case r := <-requestPlayer:
			player, playerExists := players[r.key]
			r.response<- struct {player_state; bool}{*player, playerExists}
		case m := <-playerChange:
			player, playerExists := players[m.key] 
			if !playerExists {
				continue
			}
			m.modify(player)
		case n := <-playerCreate:
			if _, exists := players[n.name]; exists {
				continue
			}
			players[n.name] = &n.player
		}
	}
}
