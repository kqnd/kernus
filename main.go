package main

import "github.com/kiev/kernus/cmd"

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
