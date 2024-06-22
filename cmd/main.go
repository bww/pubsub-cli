package main

import (
	"fmt"
	"os"

	"github.com/bww/go-util/v1/debug"
)

const stdin = "-"

func main() {
	err := app()
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(1)
	}
}

const usage = `
Usage: pubsub <command> [options]
       pubsub -h

Commands:
  topic         Manage topics.
  subscription  Manage subscriptions.
  publish       Publish messages to a topic.
  receive       Receive messages from a subscription.
  help          Display this help information.
`

func app() error {
	app := os.Args[0]

	if os.Getenv("PUBSUB_DEBUG_ROUTINES") != "" {
		fmt.Println("--> Dumping routines on ^C")
		debug.DumpRoutinesOnInterrupt()
	}

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println(usage)
		return nil
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "pub", "publish":
		return publish(app, args)
	case "pull", "receive":
		return receive(app, args)
	case "top", "topic":
		return topics(app, args)
	case "sub", "subscription":
		return subscriptions(app, args)
	case "help":
		fallthrough
	default:
		fmt.Println(usage)
	}

	return nil
}
