package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	humanize "github.com/dustin/go-humanize"
)

type Format string

const (
	None   = Format("none")
	Pretty = Format("pretty")
	JSON   = Format("json")
)

const receiveUsage = `
Usage: pubsub receive <subcommand> [options]
       pubsub receive help

Commands:
  data    Recieve message data.
  help    Display this help information.
`

func receive(cmd string, args []string) error {
	if len(args) < 1 {
		fmt.Println(receiveUsage)
		return nil
	}

	cmd, args = args[0], args[1:]
	switch cmd {
	case "data":
		return receiveData(cmd, args)
	case "help":
		fallthrough
	default:
		fmt.Println(receiveUsage)
	}

	return nil
}

func receiveData(cmd string, args []string) error {
	cmdline := newFlags(cmd)

	var (
		fCount      = cmdline.Int("count", 0, "The number of messages to receive. If count is less than one, process unlimited messages.")
		fWait       = cmdline.Duration("wait", 0, "When receiving unlimited messages, wait this duration for messages before canceling.")
		fNoAck      = cmdline.Bool("no-ack", false, "Don't acknowledge received messages.")
		fSubscr     = cmdline.String("subscription", "", "The subscription to receive messages from.")
		fConcurrent = cmdline.Int("concurrency", 1, "The maximum outstanding messages.")
		fOutput     = cmdline.String("output", string(Pretty), "The format used to output messages (none|pretty|json)")
	)

	cmdline.Parse(args)
	datafmt := Format(*fOutput)
	count := *fCount

	maxFlight := *fConcurrent
	if maxFlight < 1 {
		maxFlight = 1
	}
	maxRoutine := 1
	if maxFlight > 1 {
		maxRoutine = pubsub.DefaultReceiveSettings.NumGoroutines
	}

	if *fSubscr == "" {
		return fmt.Errorf("No subscription")
	}
	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	cxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := pubsub.NewClient(cxt, cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	sub := client.Subscription(*fSubscr)
	sub.ReceiveSettings.MaxOutstandingMessages = maxFlight
	sub.ReceiveSettings.NumGoroutines = maxRoutine
	sub.ReceiveSettings.Synchronous = true

	exists, err := sub.Exists(cxt)
	if err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("No such subscription: %s", *fSubscr)
	}

	mqueue := make(chan string)
	wqueue := make(chan string)

	var tbytes, tproc, tmsg int64
	recv := func(cxt context.Context, msg *pubsub.Message) {
		b := &strings.Builder{}

		if cmdline.Quiet || datafmt == None {
			b.WriteString(".")
		} else if datafmt == JSON {
			json.NewEncoder(b).Encode(msg)
		} else {
			h := fmt.Sprintf("%s @ %v", msg.ID, msg.PublishTime)
			fmt.Fprintln(b, h)
			if cmdline.Verbose {
				fmt.Fprintln(b, strings.Repeat("─", len(h)))
				if len(msg.Attributes) > 0 {
					mw, lw := 0, 40
					for k, _ := range msg.Attributes {
						if l := len(k); l > mw {
							mw = l
						}
					}
					if mw > lw {
						mw = lw
					}
					spec := fmt.Sprintf("%%%ds: ", mw)
					for k, v := range msg.Attributes {
						fmt.Fprintf(b, spec, k)
						fmt.Fprintln(b, v)
					}
					fmt.Fprintln(b, strings.Repeat("─", len(h)))
				}
				d := string(msg.Data)
				if l := len(d); l > 0 && d[l-1] != '\n' {
					fmt.Fprintln(b, d)
				} else {
					fmt.Fprint(b, d)
				}
				fmt.Fprintln(b, "◆")
			}
		}

		mqueue <- b.String()

		if !*fNoAck {
			msg.Ack()
		}

		atomic.AddInt64(&tbytes, int64(len(msg.Data)))
		atomic.AddInt64(&tmsg, 1)
	}

	if cmdline.Verbose {
		conf, err := sub.Config(cxt)
		if err != nil {
			return err
		}
		if count > 0 && *fWait > 0 {
			fmt.Printf("Receiving up to %d messages from %s (%s) for %v...\n", count, sub.ID(), conf.Topic.ID(), *fWait)
		} else if count > 0 {
			fmt.Printf("Receiving %d messages from %s (%s)...\n", count, sub.ID(), conf.Topic.ID())
		} else if *fWait > 0 {
			fmt.Printf("Receiving from %s (%s) for %v...\n", sub.ID(), conf.Topic.ID(), *fWait)
		} else {
			fmt.Printf("Receiving forever from %s (%s)...\n", sub.ID(), conf.Topic.ID())
		}
	}

	var wait sync.WaitGroup

	go func() {
		wait.Add(1)
		defer wait.Done()
		var deadline <-chan time.Time
		for {
			if v := *fWait; v > 0 {
				deadline = time.After(v)
			} else {
				deadline = make(chan time.Time) // will never be ready
			}
			select {
			case <-cxt.Done():
				return
			case <-deadline:
				if cmdline.Verbose {
					wqueue <- fmt.Sprintf("Canceling after receiving for %v...\n", *fWait)
				}
				cancel()
				return
			case m, ok := <-mqueue:
				if !ok {
					return
				}
				res := atomic.AddInt64(&tproc, 1)
				if count < 1 || res <= int64(count) {
					wqueue <- m
				}
				if count > 0 && res >= int64(count) {
					cancel()
					return
				}
			}
		}
	}()

	go func() {
		wait.Add(1)
		defer wait.Done()
		for {
			select {
			case <-cxt.Done():
				return
			case m, ok := <-wqueue:
				if !ok {
					return
				}
				fmt.Print(m)
			}
		}
	}()

	err = sub.Receive(cxt, recv)
	if err != nil && err != context.Canceled {
		if s, ok := status.FromError(err); !ok || s.Code() != codes.Canceled {
			return fmt.Errorf("Could not create backup: %v", err)
		}
	}

	wait.Wait()

	close(mqueue)
	close(wqueue)

	if cmdline.Quiet {
		fmt.Println()
	}

	if cmdline.Verbose {
		fmt.Printf("--> Received %d messages (%s) from %s\n", tmsg, humanize.Bytes(uint64(tbytes)), *fSubscr)
	}
	return nil
}
