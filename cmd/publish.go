package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/linkedin/goavro/v2"

	humanize "github.com/dustin/go-humanize"
)

const publishUsage = `
Usage: pubsub publish <subcommand> [options]
       pubsub publish help

Commands:
  data  	Publish raw data.
  avro  	Publish records from avros.
  help    Display this help information.
`

func publish(cmd string, args []string) error {
	if len(args) < 1 {
		fmt.Println(publishUsage)
		return nil
	}

	cmd, args = args[0], args[1:]
	switch cmd {
	case "data":
		return publishData(cmd, args)
	case "avro":
		return publishAvro(cmd, args)
	case "help":
		fallthrough
	default:
		fmt.Println(publishUsage)
	}

	return nil
}

func publishData(cmd string, args []string) error {
	var attrPairs []string
	var topicName string
	var count int

	cxt := context.Background()

	cmdline := newFlags(cmd)
	cmdline.StringVar(&topicName, "topic", "", "The topic to operate on.")
	cmdline.IntVar(&count, "count", 0, "Repeatedly publish the input message <count> times.")
	cmdline.StringSliceVar(&attrPairs, "attr", nil, "Define attribute(s) to be set on enqueued messages, specified as 'key=value'. You may provide this flag multiple times.")
	cmdline.Parse(args)

	if topicName == "" {
		return fmt.Errorf("No topic defined")
	}
	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	var attrs map[string]string
	for _, e := range attrPairs {
		if attrs == nil {
			attrs = make(map[string]string)
		}
		if x := strings.Index(e, "="); x > 0 {
			attrs[strings.TrimSpace(e[:x])] = strings.TrimSpace(e[x+1:])
		} else {
			return fmt.Errorf("Invalid attribute format: %s", e)
		}
	}

	client, err := pubsub.NewClient(cxt, cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(cxt)
	if err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("No such topic")
	}
	defer topic.Stop()

	count = max(count, 1)
	var tbytes, tmsg uint64
	for _, e := range cmdline.Args() {
		var r io.Reader
		if e != stdin {
			file, err := os.Open(e)
			if err != nil {
				return err
			}
			defer file.Close()
			r = file
		} else {
			r = os.Stdin
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		for i := 0; i < count; i++ {
			res := topic.Publish(cxt, &pubsub.Message{Attributes: attrs, Data: data})
			serverId, err := res.Get(cxt)
			if err != nil {
				return fmt.Errorf("Publish failed: %v", err)
			}
			if cmdline.Verbose {
				fmt.Printf("--> Published %s to %s (%s)\n", humanize.Bytes(uint64(len(data))), topicName, serverId)
			} else {
				fmt.Print(".")
			}
		}

		tbytes += uint64(len(data))
		tmsg++
	}
	if !cmdline.Verbose {
		fmt.Printf("\n--> Published %d messages (%s) to %s\n", tmsg, humanize.Bytes(tbytes), topicName)
	}

	return nil
}

func publishAvro(cmd string, args []string) error {
	var (
		topicName      string
		fieldId        string
		fieldTimestamp string
		attrId         string
		attrTimestamp  string
	)

	cxt := context.Background()

	cmdline := newFlags(cmd)
	cmdline.StringVar(&topicName, "topic", "", "The topic to operate on.")
	cmdline.StringVar(&fieldId, "field:id", "", "The name of the Avro record field that the publish timestamp should be taken from. The value of this field will be be set as an attribute on each message named -attr:id.")
	cmdline.StringVar(&fieldTimestamp, "field:timestamp", "", "The name of the Avro record field that the record identifier should be taken from. The value of this field will be be set as an attribute on each message named -attr:timestamp.")
	cmdline.StringVar(&attrId, "attr:id", "id", "The name of the attribute to be used for the record identifier field, if available.")
	cmdline.StringVar(&attrTimestamp, "attr:timestamp", "ts", "The name of the attribute to be used for the record timestamp field, if available.")

	cmdline.Parse(args)

	if topicName == "" {
		return fmt.Errorf("No topic defined")
	}
	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	client, err := pubsub.NewClient(cxt, cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(cxt)
	if err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("No such topic")
	}
	defer topic.Stop()

	for _, e := range cmdline.Args() {
		var r io.Reader
		if e != stdin {
			file, err := os.Open(e)
			if err != nil {
				return err
			}
			defer file.Close()
			r = file
		} else {
			r = os.Stdin
		}

		r = bufio.NewReader(r)
		ocfr, err := goavro.NewOCFReader(r)
		if err != nil {
			return err
		}

		var tbytes, tmsg uint64
		codec := ocfr.Codec()
		for ocfr.Scan() {
			item, err := ocfr.Read()
			if err != nil {
				return err
			}

			data, err := codec.BinaryFromNative(nil, item)
			if err != nil {
				return err
			}

			attrs := make(map[string]string)
			if m, ok := item.(map[string]interface{}); ok {
				if fieldId != "" {
					if v := m[fieldId]; v != nil {
						attrs[attrId] = stringer(v)
					}
				}
				if fieldTimestamp != "" {
					if v := m[fieldTimestamp]; v != nil {
						attrs[attrTimestamp] = stringer(v)
					}
				}
			}

			res := topic.Publish(cxt, &pubsub.Message{Attributes: attrs, Data: data})
			if err != nil {
				return fmt.Errorf("Could not publish: %v", err)
			}

			serverId, err := res.Get(cxt)
			if err != nil {
				return fmt.Errorf("Publish failed: %v", err)
			}

			if cmdline.Verbose {
				fmt.Printf("--> Published %s to %s (%s)\n", humanize.Bytes(uint64(len(data))), topicName, serverId)
				if len(attrs) > 0 {
					fmt.Printf("    %s\n", dumpAttrs(attrs))
				}
			} else {
				fmt.Print(".")
			}

			tbytes += uint64(len(data))
			tmsg++
		}
		if !cmdline.Verbose {
			fmt.Printf("\n--> Published %d messages (%s) to %s\n", tmsg, humanize.Bytes(tbytes), topicName)
		}
	}

	return nil
}
