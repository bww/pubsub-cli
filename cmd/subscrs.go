package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var subscriptions = &cobra.Command{
	Use:     "subscription",
	Aliases: []string{"subscriptions", "subs", "sub"},
	Short:   "Manage subscriptions",
}

func init() {
	listSubscriptions.MarkFlagRequired("project")
	subscriptions.AddCommand(listSubscriptions)

	createSubscriptions.MarkFlagRequired("project")
	createSubscriptions.MarkFlagRequired("topic")
	subscriptions.AddCommand(createSubscriptions)

	deleteSubscriptions.MarkFlagRequired("project")
	deleteSubscriptions.MarkFlagRequired("topic")
	subscriptions.AddCommand(deleteSubscriptions)
}

var listSubscriptions = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available subscriptions",
	Run: func(cmd *cobra.Command, args []string) {
		cxt := context.Background()

		client, err := pubsub.NewClient(cxt, projectName)
		cobra.CheckErr(err)
		defer client.Close()

		iter := client.Subscriptions(cxt)
		for {
			sub, err := iter.Next()
			if err == iterator.Done {
				break
			} else {
				cobra.CheckErr(err)
			}

			conf, err := sub.Config(cxt)
			cobra.CheckErr(err)

			fmt.Printf("--> %s { topic: %v, deadline: %v, retain: %v }\n", sub, conf.Topic, conf.AckDeadline, conf.RetainAckedMessages)
		}
	},
}

var createSubscriptions = &cobra.Command{
	Use:     "new",
	Aliases: []string{"create"},
	Short:   "Create a subscription",
	Run: func(cmd *cobra.Command, args []string) {
		cxt := context.Background()

		client, err := pubsub.NewClient(cxt, projectName)
		cobra.CheckErr(err)
		defer client.Close()

		topic := client.Topic(topicName)
		exists, err := topic.Exists(cxt)
		cobra.CheckErr(err)
		if !exists {
			cobra.CheckErr(fmt.Errorf("No such topic: %v", topic))
		}

		for _, e := range args {
			sub := client.Subscription(e)
			exists, err := sub.Exists(cxt)
			cobra.CheckErr(err)

			if exists {
				fmt.Printf("--> [exists] %s\n", e)
				continue
			}

			sub, err = client.CreateSubscription(cxt, e, pubsub.SubscriptionConfig{Topic: topic})
			cobra.CheckErr(err)
			fmt.Printf("--> [create] %s\n", e)
		}
	},
}

var deleteSubscriptions = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Short:   "Delete a subscription",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := pubsub.NewClient(context.Background(), projectName)
		cobra.CheckErr(err)
		defer client.Close()

		for _, e := range args {
			sub := client.Subscription(e)
			exists, err := sub.Exists(context.Background())
			cobra.CheckErr(err)

			if !exists {
				fmt.Printf("--> [missing] %s\n", e)
				continue
			}

			err = sub.Delete(context.Background())
			cobra.CheckErr(err)
			fmt.Printf("--> [deleted] %s\n", e)
		}
	},
}
