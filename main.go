/*
main.go initializes the world and starts listening for connections

*/
package main

import (
	"fmt"
	"math"
	"strconv"
)

const version = `0.0.5`
const defaultPort = 9241

type identifier int32

func (i identifier) String() string {
	return strconv.Itoa(int(i))
}

const invalidIdentifier = identifier(-1)

const endl = "\r\n"

type ChainTime uint64

func (c ChainTime) String() string {
	return strconv.Itoa(int(c))
}

var NextChainTime chan ChainTime // @todo make local, and passed to ThingManagers, which make it accessible
const NotChaining = ChainTime(0)

var NextId chan identifier // @todo make local, and passed to ThingManagers, which make it accessible
var CurrentId chan identifier

func NewWorld() *World {
	go func() {
		NextChainTime = make(chan ChainTime)
		chainTime := ChainTime(0)
		for {
			NextChainTime <- chainTime
			chainTime++
			if chainTime == math.MaxInt64 {
				panic("Max chaintime reached")
			}
		}
	}()

	go func() {
		NextId = make(chan identifier)
		CurrentId = make(chan identifier)
		id := identifier(0)
		for {
			select {
			case NextId <- id:
				//fmt.Println("debug: id provided " + id.String())
				id++
			case CurrentId <- id:
			}
		}
	}()

	initCommands()

	pm := PlayerManager(*NewThingManager())
	rm := RoomManager(*NewThingManager())
	nm := NpcManager(*NewThingManager())
	im := ItemManager(*NewThingManager())
	world := &World{
		players: &pm,
		rooms:   &rm,
		npcs:    &nm,
		items:   &im,
	}
	initDb(world)

	_, exists := RoomManager(*world.rooms).GetById(0)
	if !exists {
		fmt.Println("Creating initial room")
		ThingManager(*world.rooms).Add(&Room{
			id:          identifier(0),
			name:        "The Beginning",
			Description: "Everything has a beginning. This is only one of many beginnings you will soon find as I continue typing in order to create a wall of text to test this. It's a very long sentence that precedes this slightly shorter one. Blarglblargl.",
			Exits:       make(map[Direction]identifier),
			Players:     make(map[identifier]bool),
			Items:       make(map[identifier]PlayerItemType),
		})
	}

	return world
}

/*
func debug() {
	v8ctx := v8.NewContext()
	//	v8ctx.Eval(`this.console = { "log" : function(args) { _console_log.apply(null, arguments) }}`)
	v8ctx.Eval("var fubar = 42")
	v8ctx.AddFunc("_console_log", func(args ...interface{}) interface{} {
		fmt.Printf("Go console log: ")
		for i := 0; i < len(args); i++ {
			fmt.Println()
			return ""
		}
		return ""
	})
	ret := v8ctx.MustEval(`
        var a = 1;
        var b = 'B'
        a += 2;
        a;
        `)
	fmt.Println("Eval result:", int(ret.(float64)))
	v8ctx.Eval(`console.log(a + 'fubar'+ b + 'baz' + 'something else`)
	v8ctx.Eval(`console.log('Hello World, 198570154')`)
	v8ctx.AddFunc("func_call", func(args ...interface{}) interface{} {
		f := func(args ...interface{}) interface{} {
			return "V8"
		}
		ret, _ = args[0].(v8.V8Function).Call("Go", 2, 1, f)
		return ret
	})
	fmt.Println(v8ctx.MustEval(`
        func_call(function() {
                return 'Hello ' + arguments[0];
        })
        `).(string))

	v8ctx.AddFunc("go_println", func(args ...interface{}) interface{} {
		if len(args) == 0 {
			return nil
		}
		var argString string
		var ok bool
		if argString, ok = args[0].(string); !ok {
			return nil
		}
		if len(argString) < 3 {
			return nil
		}
		argString = argString[1 : len(argString)-2]
		fmt.Println(argString)
		fmt.Println("wha?")
		return nil
	})
	v8ctx.Eval("go_println('Go Printline Works!')")

	toEval := "var a = 42;"
	selfId := identifier(42)
	wrappedToEval := "(function(self) {" + toEval + "})(" + selfId.String() + ")"
	v8ctx.Eval(wrappedToEval)
}
*/

func main() {
	world := NewWorld()
	//	world.script.Eval("mud_println('javascript engine running');")
	fmt.Println("version " + version)
	listen(*world)
}
