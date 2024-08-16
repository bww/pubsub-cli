package main

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var topics = &cobra.Command{
	Use:     "topic",
	Aliases: []string{"topics"},
	Short:   "Manage topics",
}

func init() {
	listTopics.MarkFlagRequired("project")
	topics.AddCommand(listTopics)

	createTopic.MarkFlagRequired("project")
	topics.AddCommand(createTopic)

	deleteTopic.MarkFlagRequired("project")
	topics.AddCommand(deleteTopic)
}

var listTopics = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available topics",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := pubsub.NewClient(context.Background(), projectName)
		cobra.CheckErr(err)
		defer client.Close()

		iter := client.Topics(context.Background())
		for {
			topic, err := iter.Next()
			if err == iterator.Done {
				break
			} else {
				cobra.CheckErr(err)
			}
			logf("--> %s\n", topic)
		}
	},
}

var createTopic = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create", "make"},
	Short:   "Create a topic",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := pubsub.NewClient(context.Background(), projectName)
		cobra.CheckErr(err)
		defer client.Close()

		for _, e := range args {
			topic := client.Topic(e)
			exists, err := topic.Exists(context.Background())
			cobra.CheckErr(err)

			if exists {
				logf("--> [exists] %s\n", e)
				continue
			}

			topic, err = client.CreateTopic(context.Background(), e)
			cobra.CheckErr(err)
			logf("--> [create] %s\n", e)
		}
	},
}

var deleteTopic = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Short:   "Delete a topic",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := pubsub.NewClient(context.Background(), projectName)
		cobra.CheckErr(err)
		defer client.Close()

		for _, e := range args {
			topic := client.Topic(e)
			exists, err := topic.Exists(context.Background())
			cobra.CheckErr(err)

			if !exists {
				logf("--> [missing] %s\n", e)
				continue
			}

			err = topic.Delete(context.Background())
			cobra.CheckErr(err)
			logf("--> [deleted] %s\n", e)
		}
	},
}
