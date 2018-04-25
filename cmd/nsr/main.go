package main

import (
	"log"
	"os"

	"github.com/urfave/cli"

	"jw4.us/nsrecorder"
)

func main() {
	app := cli.NewApp()
	app.Name = "nsr"
	app.Version = nsrecorder.Version
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
