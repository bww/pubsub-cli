package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"
)

const topicsUsage = `
Usage: pubsub topics <subcommand> [options]
       pubsub topics help

Commands:
  create  Create a new topic.
  list    List topics.
  delete  Delete a topic.
  help    Display this help information.
`

func topics(cmd string, args []string) error {
	if len(args) < 1 {
		fmt.Println(topicsUsage)
		return nil
	}

	cmd, args = args[0], args[1:]
	switch cmd {
	case "ls", "list":
		return listTopics(cmd, args)
	case "mk", "create":
		return createTopic(cmd, args)
	case "rm", "delete":
		return deleteTopic(cmd, args)
	case "help":
		fallthrough
	default:
		fmt.Println(topicsUsage)
	}

	return nil
}

func listTopics(cmd string, args []string) error {
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

	iter := client.Topics(context.Background())
	for {
		topic, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}
		fmt.Printf("--> %s\n", topic)
	}

	return nil
}

func createTopic(cmd string, args []string) error {
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
		topic := client.Topic(e)

		exists, err := topic.Exists(context.Background())
		if err != nil {
			return err
		}

		if exists {
			fmt.Printf("--> [exists] %s\n", e)
			continue
		}

		topic, err = client.CreateTopic(context.Background(), e)
		if err != nil {
			return err
		}

		fmt.Printf("--> [create] %s\n", e)
	}

	return nil
}

func deleteTopic(cmd string, args []string) error {
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
		topic := client.Topic(e)

		exists, err := topic.Exists(context.Background())
		if err != nil {
			return err
		}

		if !exists {
			fmt.Printf("--> [missing] %s\n", e)
			continue
		}

		err = topic.Delete(context.Background())
		if err != nil {
			return err
		}

		fmt.Printf("--> [deleted] %s\n", e)
	}

	return nil
}
