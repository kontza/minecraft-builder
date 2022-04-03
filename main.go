package main

import (
	"os"

	"github.com/kontza/minecraft_builder/builder_application"
)

func usage() {
	println(`minecraft-builder 2.0

USAGE:
    minecraft-builder [path]

ARGS:
    <path>
        a path to Minecraft Ansible's 'group_vars/all'
`)
	os.Exit(0)
}

func init() {
	if len(os.Args) < 2 {
		usage()
	}
}

func main() {
	app := builder_application.NewApplicationBuilder()
	app.RunApplication()
}
