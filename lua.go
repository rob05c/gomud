package main

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"strconv"
)

func funcs() map[string]lua.Function {
	return map[string]lua.Function{
		"gomud_println": luaPrintln,
	}
}

func initLua(world *World) *lua.State {
	l := lua.NewState()
	lua.OpenLibraries(l)
	for name, f := range funcs() {
		l.Register(name, f)
	}
	return l
}

// luaPrintln prints a string to the Server console. Used for server debugging. NPC scripters should not call this, and it should not be in the user-facing help.
func luaPrintln(l *lua.State) int {
	n := l.Top() // Number of arguments.
	if n != 1 {
		l.PushString("incorrect number of arguments: expected 1 got " + strconv.Itoa(n))
		l.Error() // panics
	}

	s, ok := l.ToString(1)
	if !ok {
		l.PushString("incorrect argument: expected string")
		l.Error() // panics
	}

	fmt.Println("Lua says: " + s)

	return 0 // Result count
}
