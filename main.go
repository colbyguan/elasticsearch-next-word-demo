package main

import (
	"fmt"
	"os"

	"github.com/colbyguan/next-word-demo/lib"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("no args, running default")
	} else if args[0] == "populate" {
		lib.PopulateIndex()
	} else if args[0] == "bootstrap" {
		lib.CreateTextFile()
	}
}
