package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli"

	"jw4.us/nsrecorder"
)

func main() {
	app := cli.NewApp()
	app.Name = "nsr"
	app.Version = nsrecorder.Version
	app.Commands = []cli.Command{list}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var (
	topicFlag   = cli.StringFlag{Name: "topic", EnvVar: "TOPIC", Value: "dns"}
	channelFlag = cli.StringFlag{Name: "channel", EnvVar: "CHANNEL", Value: "recorder-dev"}
	lookupdFlag = cli.StringSliceFlag{Name: "lookupd", EnvVar: "LOOKUPD", Value: &cli.StringSlice{"127.0.0.1:4161"}}

	list = cli.Command{
		Name:   "list",
		Action: listAction,
		Flags:  []cli.Flag{topicFlag, channelFlag, lookupdFlag},
	}
)

func listAction(c *cli.Context) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "topic", c.String("topic"))
	ctx = context.WithValue(ctx, "channel", c.String("channel"))
	ctx = context.WithValue(ctx, "lookupd", c.StringSlice("lookupd"))

	w := nsrecorder.NewWatcher(ctx)

	<-sigChan
	cancel()
	w.Stop()

	return nil
}
