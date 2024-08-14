package main

import (
	"fmt"
	"os"
	"time"

	debugutil "github.com/bww/go-util/v1/debug"
	"github.com/spf13/cobra"
)

const stdin = "-"

func main() {
	if os.Getenv("PUBSUB_DEBUG_ROUTINES") != "" {
		fmt.Println("--> Dumping routines on ^C")
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
	quiet   bool
)

var Root = &cobra.Command{
	Use:   "pubsub",
	Short: "A CLI interface to the GCP PubSub service", Long: "A CLI interface to the GCP PubSub service",
}

func init() {
	Root.PersistentFlags().StringVar(&projectName, "project", "", "The GCP project we are operating on")
	Root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Be more verbose")
	Root.PersistentFlags().BoolVar(&debug, "debug", false, "Be extremely verbose")
	Root.PersistentFlags().BoolVar(&quiet, "quiet", false, "Be extra quiet")

	Root.AddCommand(publish)
	Root.AddCommand(receive)
	Root.AddCommand(topics)
	Root.AddCommand(subscriptions)
}
