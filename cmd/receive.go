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
	"github.com/spf13/cobra"
)

type Format string

const (
	None   = Format("none")
	Pretty = Format("pretty")
	JSON   = Format("json")
)

type message struct {
	Msg *pubsub.Message
	Dsp string
	Ack func()
}

func (m message) Mack() {
	if !noAck {
		m.Ack()
	}
}

var receive = &cobra.Command{
	Use:     "receive",
	Aliases: []string{"recv"},
	Short:   "Receive messages from a subscription",
}

func init() {
	receiveData.Flags().IntVar(&count, "count", -1, "The maximum number of messages to receive. If count is less than one, process unlimited messages.")
	receiveData.Flags().IntVar(&expect, "expect", -1, "The number of messages we expect to receive. Expect is effectively --count with an assertion. It is used in combination with --wait to assert a certain number of messages were received before the deadline.")
	receiveData.Flags().DurationVar(&wait, "wait", 0, "When receiving unlimited messages, wait this duration for messages before canceling.")
	receiveData.Flags().BoolVar(&noAck, "no-ack", false, "Don't acknowledge received messages.")
	receiveData.Flags().StringVar(&subscrName, "subscription", "", "The subscription to receive messages from.")
	receiveData.Flags().IntVar(&concurrent, "concurrency", 1, "The maximum outstanding messages.")
	receiveData.Flags().StringVar(&output, "output", string(Pretty), "The format used to output messages (none|pretty|json)")
	receiveData.MarkFlagRequired("project")
	receiveData.MarkFlagRequired("subscription")

	receive.AddCommand(receiveData)
}

var receiveData = &cobra.Command{
	Use:   "data",
	Short: "Receieve data from a subscription",
	Run: func(cmd *cobra.Command, args []string) {
		cxt := context.Background()

		datafmt := Format(output)
		concurrent = max(1, concurrent)
		routines := 1
		if concurrent > 1 {
			routines = pubsub.DefaultReceiveSettings.NumGoroutines
		}

		cxt, cancel := context.WithCancel(cxt)
		defer cancel()

		client, err := pubsub.NewClient(cxt, projectName)
		cobra.CheckErr(err)
		defer client.Close()

		sub := client.Subscription(subscrName)
		sub.ReceiveSettings.MaxOutstandingMessages = concurrent
		sub.ReceiveSettings.NumGoroutines = routines
		sub.ReceiveSettings.Synchronous = true

		exists, err := sub.Exists(cxt)
		cobra.CheckErr(err)
		if !exists {
			cobra.CheckErr(fmt.Errorf("No such subscription: %s", subscrName))
		}

		mqueue := make(chan message)
		wqueue := make(chan string)

		var tbytes, tproc, tmsg int64
		recv := func(cxt context.Context, msg *pubsub.Message) {
			b := &strings.Builder{}

			if quiet > 1 {
				// don't print anything...
			} else if quiet > 0 || datafmt == None {
				b.WriteString(".")
			} else if datafmt == JSON {
				json.NewEncoder(b).Encode(msg)
			} else {
				h := fmt.Sprintf("%s @ %v", msg.ID, msg.PublishTime)
				fmt.Fprintln(b, h)
				if verbose {
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

			mqueue <- message{
				Msg: msg,
				Dsp: b.String(),
				Ack: func() {
					msg.Ack()
					atomic.AddInt64(&tbytes, int64(len(msg.Data)))
					atomic.AddInt64(&tmsg, 1)
				},
			}
		}

		if verbose {
			conf, err := sub.Config(cxt)
			cobra.CheckErr(err)
			if expect >= 0 && wait > 0 {
				fmt.Printf("Expecting %d messages from %s (%s) for %v...\n", expect, sub.ID(), conf.Topic.ID(), wait)
			} else if count >= 0 && wait > 0 {
				fmt.Printf("Receiving up to %d messages from %s (%s) for %v...\n", count, sub.ID(), conf.Topic.ID(), wait)
			} else if expect >= 0 {
				fmt.Printf("Expected %d messages from %s (%s)...\n", expect, sub.ID(), conf.Topic.ID())
			} else if count >= 0 {
				fmt.Printf("Receiving %d messages from %s (%s)...\n", count, sub.ID(), conf.Topic.ID())
			} else if wait > 0 {
				fmt.Printf("Receiving from %s (%s) for %v...\n", sub.ID(), conf.Topic.ID(), wait)
			} else {
				fmt.Printf("Receiving forever from %s (%s)...\n", sub.ID(), conf.Topic.ID())
			}
		}

		var wg sync.WaitGroup

		go func() {
			wg.Add(1)
			defer wg.Done()
			var deadline <-chan time.Time
			for {
				if v := wait; v > 0 {
					deadline = time.After(v)
				} else {
					deadline = make(chan time.Time) // will never be ready
				}
				select {
				case <-cxt.Done():
					return
				case <-deadline:
					if verbose {
						wqueue <- fmt.Sprintf("Canceling after receiving for %v...\n", wait)
					}
					cancel()
					return
				case m, ok := <-mqueue:
					if !ok {
						return
					}
					res := atomic.AddInt64(&tproc, 1)
					switch {
					case expect >= 0:
						if res <= int64(expect) {
							m.Mack() // we've consumed the message
							wqueue <- m.Dsp
						}
						if res+1 > int64(expect) {
							cancel()
							return
						}
					case count >= 0:
						if res <= int64(count) {
							m.Mack() // we've consumed the message
							wqueue <- m.Dsp
						}
						if res+1 > int64(count) {
							cancel()
							return
						}
					default:
						m.Mack() // we've consumed the message
						wqueue <- m.Dsp
					}
				}
			}
		}()

		go func() {
			wg.Add(1)
			defer wg.Done()
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
				cobra.CheckErr(fmt.Errorf("Could not create backup: %w", err))
			}
		}

		wg.Wait()

		close(mqueue)
		close(wqueue)

		if expect >= 0 && tmsg != int64(expect) {
			cobra.CheckErr(fmt.Errorf("Expected: %d messages; received: %d", expect, tmsg))
		}
		if quiet == 1 && tmsg > 0 {
			fmt.Println()
		}
		if verbose {
			fmt.Printf("--> Received %d messages (%s) from %s\n", tmsg, humanize.Bytes(uint64(tbytes)), subscrName)
		}
	},
}