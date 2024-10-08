package main

import (
	"os"

	"github.com/hectorchu/proc/cmd"
)

func main() {
	cmd.FindReplace(os.Args[1], os.Args[2]).Std()
}
