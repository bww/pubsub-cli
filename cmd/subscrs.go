package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"
)

const subscriptionsUsage = `
Usage: pubsub subscriptions <subcommand> [options]
       pubsub subscriptions help

Commands:
  create  Create a new subscription.
  list    List subscriptions
  delete  Delete a subscription.
  help    Display this help information.
`

func subscriptions(cmd string, args []string) error {
	if len(args) < 1 {
		fmt.Println(subscriptionsUsage)
		return nil
	}

	cmd, args = args[0], args[1:]
	switch cmd {
	case "ls", "list":
		return listSubscriptions(cmd, args)
	case "mk", "create":
		return createSubscription(cmd, args)
	case "rm", "delete":
		return deleteSubscription(cmd, args)
	case "help":
		fallthrough
	default:
		fmt.Println(subscriptionsUsage)
	}

	return nil
}

func listSubscriptions(cmd string, args []string) error {
	cmdline := newFlags(cmd)
	cmdline.Parse(args)

	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	client, err := pubsub.NewClient(context.Background(), cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	iter := client.Subscriptions(context.Background())
	for {
		sub, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		conf, err := sub.Config(context.Background())
		if err != nil {
			return err
		}

		fmt.Printf("--> %s { topic: %v, deadline: %v, retain: %v }\n", sub, conf.Topic, conf.AckDeadline, conf.RetainAckedMessages)
	}

	return nil
}

func createSubscription(cmd string, args []string) error {
	var topicName string

	cmdline := newFlags(cmd)
	cmdline.StringVar(&topicName, "topic", "", "The topic to associate the subscription with.")
	cmdline.Parse(args)

	if topicName == "" {
		return fmt.Errorf("No topic")
	}
	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	client, err := pubsub.NewClient(context.Background(), cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(context.Background())
	if err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("No such topic: %v", topic)
	}

	for _, e := range cmdline.Args() {
		sub := client.Subscription(e)

		exists, err := sub.Exists(context.Background())
		if err != nil {
			return err
		}

		if exists {
			fmt.Printf("--> [exists] %s\n", e)
			continue
		}

		sub, err = client.CreateSubscription(context.Background(), e, pubsub.SubscriptionConfig{Topic: topic})
		if err != nil {
			return err
		}

		fmt.Printf("--> [create] %s\n", e)
	}

	return nil
}

func deleteSubscription(cmd string, args []string) error {
	cmdline := newFlags(cmd)
	cmdline.Parse(args)

	if cmdline.Project == "" {
		return fmt.Errorf("No project defined")
	}

	client, err := pubsub.NewClient(context.Background(), cmdline.Project)
	if err != nil {
		return err
	} else {
		defer client.Close()
	}

	for _, e := range cmdline.Args() {
		sub := client.Subscription(e)

		exists, err := sub.Exists(context.Background())
		if err != nil {
			return err
		}

		if !exists {
			fmt.Printf("--> [missing] %s\n", e)
			continue
		}

		err = sub.Delete(context.Background())
		if err != nil {
			return err
		}

		fmt.Printf("--> [deleted] %s\n", e)
	}

	return nil
}
