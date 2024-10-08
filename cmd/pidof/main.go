package main

import (
	"os"

	"github.com/hectorchu/proc/cmd"
)

func main() {
	cmd.Pidof(os.Args[1]).Std()
}
