package main

import (
	"flag"
	"fmt"
	"os"
)

type Flags struct {
	*flag.FlagSet

	Debug   bool
	Verbose bool
	Quiet   bool
	Project string

	Values struct {
		Debug   *bool
		Verbose *bool
		Quiet   *bool
		Project *string
	}
}

func newFlags(cmd string) *Flags {
	f := &Flags{
		FlagSet: flag.NewFlagSet(cmd, flag.ExitOnError),
	}
	f.Values.Debug = f.Bool("debug", false, "Be extremely verbose.")
	f.Values.Verbose = f.Bool("verbose", false, "Be more verbose.")
	f.Values.Quiet = f.Bool("quiet", false, "Only print minimal output.")
	f.Values.Project = f.String("project", os.Getenv("PUBSUB_PROJECT_ID"), "The GCP project to operate on.")
	return f
}

func (f *Flags) Parse(args []string) {
	f.FlagSet.Parse(args)
	f.Debug = *f.Values.Debug
	f.Verbose = *f.Values.Verbose
	f.Quiet = *f.Values.Quiet
	f.Project = *f.Values.Project
}

type flagList []string

func (s *flagList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func (s *flagList) String() string {
	return fmt.Sprintf("%+v", *s)
}
