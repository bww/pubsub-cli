package main

import (
	"os"
	"time"

	debugutil "github.com/bww/go-util/v1/debug"
	"github.com/spf13/cobra"
)

const stdin = "-"

func main() {
	if os.Getenv("PUBSUB_DEBUG_ROUTINES") != "" {
		logln("Dumping routines on ^C")
		debugutil.DumpRoutinesOnInterrupt()
	}
	Root.Execute()
}

var (
	projectName string
	subscrName  string
	topicName   string

	attrPairs  []string
	count      int
	expect     int
	wait       time.Duration
	noAck      bool
	concurrent int
	output     string

	debug   bool
	verbose bool
	quiet   int
)

var Root = &cobra.Command{
	Use:   "pubsub",
	Short: "A CLI interface to the GCP PubSub service", Long: "A CLI interface to the GCP PubSub service",
}

func init() {
	Root.PersistentFlags().StringVar(&projectName, "project", os.Getenv("PUBSUB_PROJECT"), "The GCP project we are operating on")
	Root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", os.Getenv("PUBSUB_VERBOSE") != "", "Be more verbose")
	Root.PersistentFlags().BoolVar(&debug, "debug", os.Getenv("PUBSUB_DEBUG") != "", "Be extremely verbose")
	Root.PersistentFlags().CountVar(&quiet, "quiet", "Be extra quiet")

	Root.AddCommand(publish)
	Root.AddCommand(receive)
	Root.AddCommand(topics)
	Root.AddCommand(subscriptions)
}
