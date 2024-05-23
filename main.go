package main

import (
	"os"

	"github.com/qiushiyan/degit/cmd"
)

func setDefaultCommandIfNonePresent() {
	if len(os.Args) > 1 && os.Args[1] != "--help" && os.Args[1] != "-h" {
		potentialCommand := os.Args[1]
		for _, command := range cmd.Subcommands() {
			if command == potentialCommand {
				return
			}
		}
		os.Args = append([]string{os.Args[0], "clone"}, os.Args[1:]...)
	}

}

func main() {
	setDefaultCommandIfNonePresent()
	cmd.Execute()
}
