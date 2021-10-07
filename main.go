package main

import (
	"fmt"
	"os"

	"github.com/colbyguan/next-word-demo/lib"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "serve" {
		fmt.Println("got no args, running server")
		lib.NewServer().Start()
	} else if args[0] == "populate" {
		lib.PopulateIndex()
	} else if args[0] == "bootstrap" {
		lib.CreateTextFile()
	}
}
