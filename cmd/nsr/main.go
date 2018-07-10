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
	dbFlag      = cli.StringFlag{Name: "db", EnvVar: "DB_FILE", Value: "nsr.db"}
	verboseFlag = cli.BoolFlag{Name: "verbose", EnvVar: "VERBOSE"}

	list = cli.Command{
		Name:   "list",
		Action: listAction,
		Flags:  []cli.Flag{topicFlag, channelFlag, lookupdFlag, dbFlag},
	}
)

func listAction(c *cli.Context) error {
	for _, name := range c.FlagNames() {
		log.Printf("Flag %q: %t", name, c.IsSet(name))
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "topic", c.String("topic"))
	ctx = context.WithValue(ctx, "channel", c.String("channel"))
	ctx = context.WithValue(ctx, "lookupd", c.StringSlice("lookupd"))

	store := nsrecorder.NewSQLiteStore(c.String("db"))
	if c.Bool("verbose") {
		store = nsrecorder.MultiStore(store, nsrecorder.NewLogStore())
	}
	w := nsrecorder.NewWatcher(ctx, store)

	<-sigChan
	cancel()
	w.Stop()

	return nil
}
