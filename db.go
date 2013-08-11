package main

import (
	"database/sql"
	"fmt"
	"strconv"
	_ "github.com/mattn/go-sqlite3"
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

func loadNpcs(db *sql.DB, world *metaManager) {
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

func loadItem(rows *sql.Rows, world *metaManager) {
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

func loadItems(db *sql.DB, world *metaManager) {
	fmt.Print("loading items...")
	rows, err := db.Query(`select id, name, brief, location, location_type from items;`)
	if err != nil {
		fmt.Print("dberr loadItems ")
		fmt.Println(err)
		return
	}
	defer rows.Close()
	fmt.Print("loading items...")
	for rows.Next() {
		fmt.Print("loading item")
		loadItem(rows, world)
	}
}

func itemSaver(db *sql.DB, items ItemManager) {
	addStmt, err := db.Prepare(`insert into items (id, name, brief, location, location_type) values (?,?,?,?,?);`);
	if err != nil {
		fmt.Print("dberr itemSaver 0 ")
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update items set name = ?, brief = ?, location = ?, location_type = ? where id = ?;`);
	if err != nil {
		fmt.Print("dberr itemSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from items where id = ?;`);
	if err != nil {
		fmt.Print("dberr itemSaver 2 ")
		fmt.Println(err)
		return
	}
	for {
		select {
		case t := <- ThingManager(items).saver.add:
			fmt.Println("saver item adding")
			item := t.(*Item)
			addStmt.Exec(item.id, item.name, item.brief, int(item.Location), int(item.LocationType))
			fmt.Println("saver item added")
		case changeThing := <- ThingManager(items).saver.change:
			i := changeThing.(*Item)
			fmt.Println("saver item changing " + i.id.String() + " " + i.Location.String())
			changeStmt.Exec(i.name, i.brief, int(i.Location), int(i.LocationType), int(i.id))
			fmt.Println("saver item changed " + i.id.String())
		case id := <- ThingManager(items).saver.del:
			fmt.Println("saver item deleting")
			delStmt.Exec(id)
			fmt.Println("saver item deleted")
		}
	}
}


func npcSaver(db *sql.DB, npcs NpcManager) {
	addStmt, err := db.Prepare(`insert into npcs (id, name, brief, dna, location, location_type) values (?,?,?,?,?,?);`);
	if err != nil {
		fmt.Print("dberr npcSaver 0 ")
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update npcs set name = ?, brief = ?, dna = ?, location = ?, location_type = ? where id = ?;`);
	if err != nil {
		fmt.Print("dberr npcSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from npcs where id = ?;`);
	if err != nil {
		fmt.Print("dberr npcSaver 2 ")
		fmt.Println(err)
		return
	}
	for {
		select {
		case t := <- ThingManager(npcs).saver.add:
			fmt.Println("saver npc adding")
			npc := t.(*Npc)
			addStmt.Exec(npc.id, npc.name, npc.Brief, npc.Dna, npc.Location, npc.LocationType)
			fmt.Println("saver npc added")
		case t := <- ThingManager(npcs).saver.change:
			fmt.Println("saver npc changing")
			npc := t.(*Npc)
			changeStmt.Exec(npc.name, npc.Brief, npc.Dna, npc.Location, npc.LocationType, npc.id)
			fmt.Println("saver npc changed")
		case id := <- ThingManager(npcs).saver.del:
			fmt.Println("saver npc deleting")
			delStmt.Exec(id)
			fmt.Println("saver npc deleted")
		}
	}
}

func playerSaver(db *sql.DB, players PlayerManager) {
	addStmt, err := db.Prepare(`insert into players (id, name, salt, pass, level, health, mana, room_id) values (?,?,?,?,?,?,?,?);`);
	if err != nil {
		fmt.Print("dberr playerSaver 0 ")
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update players set name = ?, salt = ?, pass = ?, level = ?, health = ?, mana = ?, room_id = ? where id = ?;`);
	if err != nil {
		fmt.Print("dberr playerSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from players where id = ?;`);
	if err != nil {
		fmt.Print("dberr playerSaver 2 ")
		fmt.Println(err)
		return
	}
	fmt.Println("playerSaver listening...")
	for {
		select {
		case t := <- ThingManager(players).saver.add:
			player := t.(*Player)
			fmt.Println("saver adding player " + player.name)
			addStmt.Exec(player.id, player.name, string(player.passthesalt), string(player.pass), player.level, player.health, player.mana, player.Room)
			fmt.Println("saver added player " + player.name)
		case t := <- ThingManager(players).saver.change:
			fmt.Println("saver changing player")
			player := t.(*Player)
			changeStmt.Exec(player.name, player.passthesalt, player.pass, player.level, player.health, player.mana, player.Room, player.id)
			fmt.Println("saver changed player")
		case id := <- ThingManager(players).saver.del:
			fmt.Println("saver deleting player")
			delStmt.Exec(id)
			fmt.Println("saver deleted player")
		}
	}
}

func roomSaver(db *sql.DB, rooms RoomManager) {
	addStmt, err := db.Prepare(`insert into rooms (id, name, description) values (?,?,?);`)
	if err != nil {
		fmt.Print("dberr roomSaver 0 ")
		fmt.Println(err)
		return
	}
	changeStmt, err := db.Prepare(`update rooms set name = ?, description = ? where id = ?;`)
	if err != nil {
		fmt.Print("dberr roomSaver 1 ")
		fmt.Println(err)
		return
	}
	delStmt, err := db.Prepare(`delete from rooms where id = ?;`);
	if err != nil {
		fmt.Print("dberr roomSaver 2 ")
		fmt.Println(err)
		return
	}

	addExitsStmt, err := db.Prepare(`insert into room_exits (id, link, direction) values (?,?,?);`)
	if err != nil {
		fmt.Print("dberr roomSaver 3 ")
		fmt.Println(err)
		return
	}
	delExitsStmt, err := db.Prepare(`delete from room_exits where id = ?;`)
	if err != nil {
		fmt.Print("dberr roomSaver 4 ")
		fmt.Println(err)
		return
	}
	for {
		select {
		case t := <- ThingManager(rooms).saver.add:
			fmt.Println("saver adding room")
			room := t.(*Room)
			addStmt.Exec(room.id, room.name, room.Description)
			fmt.Println("saver addingg room")
			for dir, link := range room.Exits {
				addExitsStmt.Exec(room.id, link, dir)
			}
			fmt.Println("saver added room")
		case t := <- ThingManager(rooms).saver.change:
			fmt.Println("saver changing room")
			room := t.(*Room)
			changeStmt.Exec(room.name, room.Description, room.id)
			delExitsStmt.Exec(room.id) /// @todo delete and recreate exits atomically
			for dir, link := range room.Exits {
				addExitsStmt.Exec(room.id, link, dir)
			}
			fmt.Println("saver changed room ")
		case id := <- ThingManager(rooms).saver.del:
			fmt.Println("saver deleting room")
			delStmt.Exec(id)
			delExitsStmt.Exec(id)
			fmt.Println("saver deleted room")
		}
	}
}

func tryLoadPlayer(world *metaManager) bool {
	if world.db == nil {
		return false
	}
	rows, err := world.db.Query(`select id, name, salt, pass, level, health, mana, room_id from players;`)
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
		Items: make(map[identifier]PlayerItemType),
	}
	rows.Scan(&player.id, &player.name, &player.passthesalt, &player.pass, &player.level, player.health, player.mana, player.Room)

	itemRows, err := world.db.Query(`select id, name, brief from items where location = ` + player.id.String() + `;`)
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

func initDb(world *metaManager) {
	db, err := sql.Open("sqlite3", "./gomud.sqlite")
	if err != nil {
		fmt.Print("dberr init ")
		fmt.Println(err)
		return
	}
	checkSchema(db)

	loadRooms(db, *world.rooms)
	loadNpcs(db, world)
	loadItems(db, world)
	setNextId(db)

	go roomSaver(db, *world.rooms)
	go itemSaver(db, *world.items)
	go npcSaver(db, *world.npcs)
	go playerSaver(db, *world.players)

	world.db = db
}
