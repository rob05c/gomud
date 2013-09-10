/*
db.go handles database/persistence

initDb() creates and assigns the db to world,
and launches saver goroutines.

These saver goroutines listen to the ThingManagers' saver chans,
and save to the db when a change is made.
*/
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
)

/// @todo ? move this to a utils file ?
func IntMax(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func checkSchema(db *sql.DB) {
	sqls := []string{
		//		`create table if not exists things (id integer, name text)`
		//		`create table if not exists containers (id integer, )`
		`create table if not exists rooms (id integer, name text, description text);`,
		`create table if not exists room_exits (id integer, link integer, direction integer);`,
		`create table if not exists items (id integer, name text, brief text, location integer, location_type integer);`,
		`create table if not exists npcs (id integer, name text, brief text, dna text, location integer, location_type integer);`,
		`create table if not exists players (id integer, name text, salt text, pass text, level integer, health integer, mana integer, room_id integer);`,
	}

	for _, sql := range sqls {
		_, err := db.Exec(sql)
		if err != nil {
			fmt.Print("dberr checkSchema ")
			fmt.Println(err)
		}
	}
}

func loadRooms(db *sql.DB, rooms RoomManager) {
	rows, err := db.Query(`select id, name, description from rooms;`)
	if err != nil {
		fmt.Print("dberr loadRooms ")
		fmt.Println(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		var description string
		rows.Scan(&id, &name, &description)
		room := Room{
			id:          identifier(id),
			name:        name,
			Description: description,
			Exits:       make(map[Direction]identifier),
			Players:     make(map[identifier]bool),
			Items:       make(map[identifier]PlayerItemType),
		}
		exitRows, err := db.Query(`select link, direction from room_exits where id = ` + room.id.String() + `;`)
		if err != nil {
			fmt.Print("dberr loadRooms ")
			fmt.Println(err)
			continue
		}
		for exitRows.Next() {
			var link int
			var dir int
			exitRows.Scan(&link, &dir)
			room.Exits[Direction(dir)] = identifier(link)
		}
		ThingManager(rooms).DbAdd(&room)
	}
}

func loadNpcs(db *sql.DB, world *World) {
	rows, err := db.Query(`select id, name, brief, dna, location, location_type from npcs;`)
	if err != nil {
		fmt.Print("dberr loadNpcs ")
		fmt.Println(err)
		return
	}
	defer rows.Close()

	var owners []struct {
		owner identifier
		ownee identifier
	}
	for rows.Next() {
		npc := Npc{
			Sleeping: false,
			Items:    make(map[identifier]bool),
		}
		rows.Scan(&npc.id, &npc.name, &npc.Brief, &npc.Dna, &npc.Location, &npc.LocationType)
		switch npc.LocationType {
		case ilRoom:
			success := world.rooms.ChangeById(npc.id, func(loc *Room) {
				loc.Items[npc.id] = piItem
			})
			if !success {
				continue
			}
		case ilPlayer:
			success := world.players.ChangeById(npc.id, func(loc *Player) {
				loc.Items[npc.id] = piItem
			})
			if !success {
				continue
			}
		case ilNpc:
			owners = append(owners, struct {
				owner identifier
				ownee identifier
			}{owner: npc.Location, ownee: npc.id})
		default:
			fmt.Println("loadNpcs got invalid item")
		}
		ThingManager(*world.npcs).DbAdd(&npc)
		for _, pair := range owners {
			world.npcs.ChangeById(pair.owner, func(loc *Npc) {
				loc.Items[pair.ownee] = true
			})
		}
	}
}

func loadItem(rows *sql.Rows, world *World) {
	item := Item{
		Items: make(map[identifier]bool),
	}
	rows.Scan(&item.id, &item.name, &item.brief, &item.Location, &item.LocationType)
	fmt.Println("loading " + item.id.String())
	switch item.LocationType {
	case ilRoom:
		success := world.rooms.ChangeById(item.Location, func(loc *Room) {
			loc.Items[item.id] = piItem
		})
		if !success {
			fmt.Println("loaditem failed on room for " + item.id.String())
			return
		}
	case ilPlayer:
		success := world.players.ChangeById(item.Location, func(loc *Player) {
			loc.Items[item.id] = piItem
		})
		if !success {
			fmt.Println("loaditem failed on player for " + item.id.String())
			return
		}
	case ilNpc:
		success := world.npcs.ChangeById(item.Location, func(loc *Npc) {
			loc.Items[item.id] = false
		})
		if !success {
			fmt.Println("loaditem failed on npc for " + item.id.String())
			return
		}
	}
	ThingManager(*world.items).DbAdd(&item)
}

func loadItems(db *sql.DB, world *World) {
	rows, err := db.Query(`select id, name, brief, location, location_type from items;`)
	if err != nil {
		fmt.Print("dberr loadItems ")
		fmt.Println(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		loadItem(rows, world)
	}
}

func itemSaver(db *sql.DB, items ItemManager) {
	addStmt, err := db.Prepare(`insert into items (id, name, brief, location, location_type) values (?,?,?,?,?);`)
	if err != nil {
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update items set name = ?, brief = ?, location = ?, location_type = ? where id = ?;`)
	if err != nil {
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from items where id = ?;`)
	if err != nil {
		fmt.Println(err)
		return
	}
	saver := ThingManager(items).saver
	for {
		select {
		case t := <-saver.add:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txAdd := tx.Stmt(addStmt)

			item := t.(*Item)
			txAdd.Exec(item.id, item.name, item.brief, int(item.Location), int(item.LocationType))

			doCommit <- tx
		case t := <-saver.change:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txChange := tx.Stmt(changeStmt)

			item := t.(*Item)
			txChange.Exec(item.name, item.brief, int(item.Location), int(item.LocationType), int(item.id))

			doCommit <- tx
		case id := <-saver.del:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txDel := tx.Stmt(delStmt)

			txDel.Exec(id)

			doCommit <- tx
		}
	}
}

func npcSaver(db *sql.DB, npcs NpcManager) {
	addStmt, err := db.Prepare(`insert into npcs (id, name, brief, dna, location, location_type) values (?,?,?,?,?,?);`)
	if err != nil {
		fmt.Print("dberr npcSaver 0 ")
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update npcs set name = ?, brief = ?, dna = ?, location = ?, location_type = ? where id = ?;`)
	if err != nil {
		fmt.Print("dberr npcSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from npcs where id = ?;`)
	if err != nil {
		fmt.Print("dberr npcSaver 2 ")
		fmt.Println(err)
		return
	}
	saver := ThingManager(npcs).saver
	for {
		select {
		case t := <-saver.add:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txAdd := tx.Stmt(addStmt)

			npc := t.(*Npc)
			txAdd.Exec(npc.id, npc.name, npc.Brief, npc.Dna, npc.Location, npc.LocationType)

			doCommit <- tx
		case t := <-saver.change:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txChange := tx.Stmt(changeStmt)

			npc := t.(*Npc)
			txChange.Exec(npc.name, npc.Brief, npc.Dna, npc.Location, npc.LocationType, npc.id)

			doCommit <- tx
		case id := <-saver.del:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txDel := tx.Stmt(delStmt)

			txDel.Exec(id)

			doCommit <- tx
		}
	}
}

func playerSaver(db *sql.DB, players PlayerManager) {
	addStmt, err := db.Prepare(`insert into players (id, name, salt, pass, level, health, mana, room_id) values (?,?,?,?,?,?,?,?);`)
	if err != nil {
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update players set name = ?, salt = ?, pass = ?, level = ?, health = ?, mana = ?, room_id = ? where id = ?;`)
	if err != nil {
		fmt.Print("dberr playerSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from players where id = ?;`)
	if err != nil {
		fmt.Print("dberr playerSaver 2 ")
		fmt.Println(err)
		return
	}
	saver := ThingManager(players).saver
	for {
		select {
		case t := <-saver.add:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txAdd := tx.Stmt(addStmt)

			player := t.(*Player)
			txAdd.Exec(player.id, player.name, string(player.passthesalt), string(player.pass), player.level, player.health, player.mana, player.Room)

			doCommit <- tx
		case t := <-saver.change:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				panic(err) // debug
				return
			}
			txChange := tx.Stmt(changeStmt)

			player := t.(*Player)
			txChange.Exec(player.name, player.passthesalt, player.pass, player.level, player.health, player.mana, player.Room, player.id)

			doCommit <- tx
		case id := <-saver.del:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				panic(err) // debug
				return
			}
			txDel := tx.Stmt(delStmt)

			txDel.Exec(id)

			doCommit <- tx
		}
	}
}

/// @note this probably isn't useful, as we often need to execute a number of statements before committing.
func doTransaction(db *sql.DB, statement *sql.Stmt, args ...interface{}) {
	tx, err := db.Begin()
	if err != nil {
		fmt.Println(err)
		panic(err) // debug
		return
	}
	txStmt := tx.Stmt(statement)

	txStmt.Exec(args...)

	doCommit <- tx
}

func roomSaver(db *sql.DB, rooms RoomManager) {
	addStmt, err := db.Prepare(`insert into rooms (id, name, description) values (?,?,?);`)
	if err != nil {
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update rooms set name = ?, description = ? where id = ?;`)
	if err != nil {
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from rooms where id = ?;`)
	if err != nil {
		fmt.Println(err)
		return
	}
	delExitsStmt, err := db.Prepare(`delete from room_exits where id = ?;`)
	if err != nil {
		fmt.Println(err)
		return
	}
	addExitsStmt, err := db.Prepare(`insert into room_exits (id, link, direction) values (?,?,?);`)
	if err != nil {
		fmt.Println(err)
		return
	}
	saver := ThingManager(rooms).saver
	for {
		select {
		case t := <-saver.add:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txAdd := tx.Stmt(addStmt)
			txAddExits := tx.Stmt(addExitsStmt)

			room := t.(*Room)
			txAdd.Exec(room.id, room.name, room.Description)
			for dir, link := range room.Exits {
				txAddExits.Exec(room.id, link, dir)
			}

			doCommit <- tx
		case t := <-saver.change:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txChange := tx.Stmt(changeStmt)
			txAddExits := tx.Stmt(addExitsStmt)
			txDelExits := tx.Stmt(delExitsStmt)
			room := t.(*Room)

			txChange.Exec(room.name, room.Description, room.id)
			txDelExits.Exec(room.id) /// @todo delete and recreate exits atomically
			for dir, link := range room.Exits {
				txAddExits.Exec(room.id, link, dir)
			}

			doCommit <- tx
		case id := <-saver.del:
			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
				continue
			}
			txDel := tx.Stmt(delStmt)
			txDelExits := tx.Stmt(delExitsStmt)

			txDel.Exec(id)
			txDelExits.Exec(id)

			doCommit <- tx
		}
	}
}

/// @todo fix loading the player's items
func tryLoadPlayer(name string, world *World) bool {
	if world.db == nil {
		return false
	}
	rows, err := world.db.Query(`select id, salt, pass, level, health, mana, room_id from players where name = '` + name + `';`)
	if err != nil {
		fmt.Print("dberr tryLoadPlayer ")
		fmt.Println(err)
		return false
	}
	defer rows.Close()
	if !rows.Next() {
		return false
	}
	player := Player{
		name:  name,
		Items: make(map[identifier]PlayerItemType),
	}
	rows.Scan(&player.id, &player.passthesalt, &player.pass, &player.level, &player.health, &player.mana, &player.Room)

	ThingManager(*world.players).DbAdd(&player)
	world.rooms.ChangeById(player.Room, func(r *Room) {
		r.Players[player.Id()] = true
	})

	itemRows, err := world.db.Query(`select id, name, brief, location, location_type from items where location = ` + player.id.String() + `;`)
	defer itemRows.Close()
	for itemRows.Next() {
		loadItem(itemRows, world)
	}

	return true
}

func setNextId(db *sql.DB) {
	tables := []string{
		`items`,
		`npcs`,
		`rooms`,
		`players`,
	}
	var maxid int
	for _, table := range tables {
		rows, err := db.Query(`select max(id) from ` + table + `;`)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer rows.Close()
		if rows.Next() {
			var id int
			rows.Scan(&id)
			if id > maxid {
				maxid = id
			}
		}
	}
	currentId := <-CurrentId
	fmt.Println("initial current id: " + strconv.Itoa(int(currentId)))
	fmt.Println("max id: " + strconv.Itoa(int(maxid)))
	for int(currentId) < maxid {
		currentId = <-NextId
	}
	fmt.Println("current id: " + strconv.Itoa(int(currentId)))
}

// sqlite commits must be sequential
var doCommit chan *sql.Tx

func commitManager() {
	for {
		tx := <-doCommit
		err := tx.Commit()
		if err != nil {
			fmt.Print("db commit err: ")
			fmt.Println(err)
			panic(err) // debug
		}
	}
}

func initDb(world *World) {
	db, err := sql.Open("sqlite3", "./gomud.sqlite")
	if err != nil {
		fmt.Print("dberr init ")
		fmt.Println(err)
		return
	}
	checkSchema(db)

	doCommit = make(chan *sql.Tx, 1000)
	loadRooms(db, *world.rooms)
	loadNpcs(db, world)
	loadItems(db, world)
	setNextId(db)

	go commitManager()
	go roomSaver(db, *world.rooms)
	go itemSaver(db, *world.items)
	go npcSaver(db, *world.npcs)
	go playerSaver(db, *world.players)

	world.db = db
}
