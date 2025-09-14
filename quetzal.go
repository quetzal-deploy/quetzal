package main

import "github.com/quetzal-deploy/quetzal/internal/cmd"

// vars from ldflags
var assets string
var version string

func main() {
	if assets == "" {
		panic("Quetzal must be compiled with \"-ldflags=-X main.assets=..\" pointing to the assets dir from the source of Quetzal.")
	}

	cmd.Execute(version, assets)
}
