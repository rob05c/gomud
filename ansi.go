package main

type colorcode string

const (
	Black     = "\x1b[30;22m"
	Darkred   = "\x1b[31;22m"
	Darkgreen = "\x1b[32;22m"
	Brown     = "\x1b[33;22m"
	Darkblue  = "\x1b[34m;22m"
	Darkpink  = "\x1b[35m;22m"
	Darkcyan  = "\x1b[36m;22m"
	Grey      = "\x1b[37m;22m"
	Darkgrey  = "\x1b[30m;1m"
	Red       = "\x1b[40;31;1m"
	Green     = "\x1b[40;32;1m"
	Yellow    = "\x1b[40;33;1m"
	Blue      = "\x1b[34;1m"
	Pink      = "\x1b[35;1m"
	Cyan      = "\x1b[36;1m"
	White     = "\x1b[37;1m"
	Reset     = "\x1b[0m"
)
