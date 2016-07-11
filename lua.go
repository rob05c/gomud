package main

import (
	"github.com/Shopify/go-lua"
)

func initLua(world *World) *lua.State {
	l := lua.NewState()
	lua.OpenLibraries(l)
	return l
}
