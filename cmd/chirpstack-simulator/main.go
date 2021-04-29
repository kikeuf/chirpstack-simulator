package main

import "github.com/kikeuf/lorawan/cmd/chirpstack-simulator/cmd"

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
