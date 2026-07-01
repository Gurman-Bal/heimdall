package main

import (
	"fmt"

	"heimdall/internal/core"
	"heimdall/internal/plugins/minecraft"
)

func main() {

	plugins := []core.Plugin{
		minecraft.New(),
	}

	for _, p := range plugins {
		fmt.Println("Loaded:", p.Name())
	}
}
