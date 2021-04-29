package main

import "github.com/kikeuf/chirpstack-simulator-2/cmd/chirpstack-simulator/cmd"

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
