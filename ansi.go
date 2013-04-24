package main

type colorcode string

const (
	Black     = "\x1b[0;30m"
	Darkred   = "\x1b[0;31m"
	Darkgreen = "\x1b[0;32m"
	Brown     = "\x1b[0;33m"
	Darkblue  = "\x1b[0;34m"
	Darkpink  = "\x1b[0;35m"
	Darkcyan  = "\x1b[0;36m"
	Grey      = "\x1b[0;37m"
	Darkgrey  = "\x1b[1;30m"
	Red       = "\x1b[1;31m"
	Green     = "\x1b[1;32m"
	Yellow    = "\x1b[1;33m"
	Blue      = "\x1b[1;34m"
	Pink      = "\x1b[1;35m"
	Cyan      = "\x1b[1;36m"
	White     = "\x1b[0;30m"
	Reset     = "\x1b[0m"
)
