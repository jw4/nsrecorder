package main

import (
	"context"
	"fmt"
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
	listFlags   = []cli.Flag{topicFlag, channelFlag, lookupdFlag, dbFlag, verboseFlag}

	list = cli.Command{
		Name:   "list",
		Action: listAction,
		Flags:  listFlags,
	}
)

func listAction(c *cli.Context) error {
	reportContext(c, listFlags)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "topic", c.String("topic"))
	ctx = context.WithValue(ctx, "channel", c.String("channel"))
	ctx = context.WithValue(ctx, "lookupd", c.StringSlice("lookupd"))

	store, err := nsrecorder.NewSQLiteStore(c.String("db"))
	if err != nil {
		return err
	}
	if c.Bool("verbose") {
		store = nsrecorder.MultiStore(store, nsrecorder.NewLogStore())
	}
	w := nsrecorder.NewWatcher(ctx, store)

	<-sigChan
	cancel()
	w.Stop()

	return nil
}

func reportContext(c *cli.Context, flags []cli.Flag) {
	log.Printf("Version: %s", nsrecorder.Version)
	for _, flag := range flags {
		log.Print(stringFlag(c, flag))
	}
}

func stringFlag(c *cli.Context, flag cli.Flag) string {
	switch v := flag.(type) {
	case cli.BoolFlag:
		return fmt.Sprintf("\t%10s: %t", flag.GetName(), c.Bool(flag.GetName()))
	case cli.StringFlag:
		return fmt.Sprintf("\t%10s: %s", flag.GetName(), c.String(flag.GetName()))
	case cli.StringSliceFlag:
		return fmt.Sprintf("\t%10s: %v", flag.GetName(), c.StringSlice(flag.GetName()))
	default:
		return fmt.Sprintf("\t%10s: (%T) %v", flag.GetName(), v, v)
	}
}
