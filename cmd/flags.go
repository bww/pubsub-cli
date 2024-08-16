package main

import (
	"os"

	flag "github.com/spf13/pflag"
)

type Flags struct {
	*flag.FlagSet

	Debug   bool
	Verbose bool
	Quiet   bool
	Project string
}

func newFlags(cmd string) *Flags {
	f := &Flags{
		FlagSet: flag.NewFlagSet(cmd, flag.ExitOnError),
	}
	f.BoolVar(&f.Debug, "debug", false, "Be extremely verbose.")
	f.BoolVar(&f.Verbose, "verbose", false, "Be more verbose.")
	f.BoolVar(&f.Quiet, "quiet", false, "Only print minimal output.")
	f.StringVar(&f.Project, "project", os.Getenv("PUBSUB_PROJECT_ID"), "The GCP project to operate on.")
	return f
}
