package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/linkedin/goavro/v2"
	"github.com/spf13/cobra"

	humanize "github.com/dustin/go-humanize"
)

var publish = &cobra.Command{
	Use:     "publish",
	Aliases: []string{"pub"},
	Short:   "Publish messages to a topic",
}

func init() {
	publishData.Flags().StringVar(&topicName, "topic", "", "The topic to operate on.")
	publishData.Flags().IntVar(&count, "count", 0, "Repeatedly publish the input message <count> times.")
	publishData.Flags().StringSliceVar(&attrPairs, "attr", nil, "Define attribute(s) to be set on enqueued messages, specified as 'key=value'. You may provide this flag multiple times.")
	publishData.MarkFlagRequired("project")
	publishData.MarkFlagRequired("topic")
	publish.AddCommand(publishData)

	publishAvro.Flags().StringVar(&topicName, "topic", "", "The topic to operate on.")
	publishAvro.Flags().StringVar(&fieldId, "field:id", "", "The name of the Avro record field that the publish timestamp should be taken from. The value of this field will be be set as an attribute on each message named -attr:id.")
	publishAvro.Flags().StringVar(&fieldTimestamp, "field:timestamp", "", "The name of the Avro record field that the record identifier should be taken from. The value of this field will be be set as an attribute on each message named -attr:timestamp.")
	publishAvro.Flags().StringVar(&attrId, "attr:id", "id", "The name of the attribute to be used for the record identifier field, if available.")
	publishAvro.Flags().StringVar(&attrTimestamp, "attr:timestamp", "ts", "The name of the attribute to be used for the record timestamp field, if available.")
	publishAvro.MarkFlagRequired("project")
	publishAvro.MarkFlagRequired("topic")
	publish.AddCommand(publishAvro)
}

var publishData = &cobra.Command{
	Use:   "data",
	Short: "Publish data to a topic",
	Run: func(cmd *cobra.Command, args []string) {
		cxt := context.Background()

		if len(args) == 0 {
			cobra.CheckErr("No data provided; specify files to publish as arguments; the file name '-' refers to STDIN")
		}

		var attrs map[string]string
		for _, e := range attrPairs {
			if attrs == nil {
				attrs = make(map[string]string)
			}
			if x := strings.Index(e, "="); x > 0 {
				attrs[strings.TrimSpace(e[:x])] = strings.TrimSpace(e[x+1:])
			} else {
				cobra.CheckErr(fmt.Errorf("Invalid attribute format: %s", e))
			}
		}

		client, err := pubsub.NewClient(cxt, projectName)
		cobra.CheckErr(err)
		defer client.Close()

		topic := client.Topic(topicName)
		exists, err := topic.Exists(cxt)
		cobra.CheckErr(err)
		if !exists {
			cobra.CheckErr("No such topic")
		}
		defer topic.Stop()

		count = max(count, 1)
		var tbytes, tmsg, dots uint64
		for _, e := range args {
			var r io.Reader
			if e != stdin {
				file, err := os.Open(e)
				cobra.CheckErr(err)
				defer file.Close()
				r = file
			} else {
				r = os.Stdin
			}

			data, err := io.ReadAll(r)
			cobra.CheckErr(err)

			for i := 0; i < count; i++ {
				res := topic.Publish(cxt, &pubsub.Message{Attributes: attrs, Data: data})
				serverId, err := res.Get(cxt)
				if err != nil {
					cobra.CheckErr(fmt.Errorf("Publish failed: %v", err))
				}
				if verbose {
					logf("Published %s to %s (%s)\n", humanize.Bytes(uint64(len(data))), topicName, serverId)
				} else if quiet == 0 {
					log(".")
					dots++
				}
			}

			tbytes += uint64(len(data))
			tmsg++
		}
		if quiet > 0 && dots > 0 {
			logln()
		}
		if !verbose {
			logf("Published %d messages (%s) to %s\n", tmsg, humanize.Bytes(tbytes), topicName)
		}
	},
}

var (
	fieldId        string
	fieldTimestamp string
	attrId         string
	attrTimestamp  string
)
var publishAvro = &cobra.Command{
	Use:   "avro",
	Short: "Publish Avros to a topic",
	Run: func(cmd *cobra.Command, args []string) {
		cxt := context.Background()

		client, err := pubsub.NewClient(cxt, projectName)
		cobra.CheckErr(err)
		defer client.Close()

		topic := client.Topic(topicName)
		exists, err := topic.Exists(cxt)
		cobra.CheckErr(err)
		if !exists {
			cobra.CheckErr("No such topic")
		}
		defer topic.Stop()

		for _, e := range args {
			var r io.Reader
			if e != stdin {
				file, err := os.Open(e)
				cobra.CheckErr(err)
				defer file.Close()
				r = file
			} else {
				r = os.Stdin
			}

			r = bufio.NewReader(r)
			ocfr, err := goavro.NewOCFReader(r)
			cobra.CheckErr(err)

			var tbytes, tmsg, dots uint64
			codec := ocfr.Codec()
			for ocfr.Scan() {
				item, err := ocfr.Read()
				cobra.CheckErr(err)

				data, err := codec.BinaryFromNative(nil, item)
				cobra.CheckErr(err)

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
					cobra.CheckErr(fmt.Errorf("Could not publish: %v", err))
				}

				serverId, err := res.Get(cxt)
				if err != nil {
					cobra.CheckErr(fmt.Errorf("Publish failed: %v", err))
				}

				if verbose {
					logf("Published %s to %s (%s)\n", humanize.Bytes(uint64(len(data))), topicName, serverId)
					if len(attrs) > 0 {
						logf("    %s\n", dumpAttrs(attrs))
					}
				} else {
					log(".")
					dots++
				}

				tbytes += uint64(len(data))
				tmsg++
			}
			if quiet > 0 && dots > 0 {
				logln()
			}
			if !verbose {
				logf("Published %d messages (%s) to %s\n", tmsg, humanize.Bytes(tbytes), topicName)
			}
		}
	},
}
