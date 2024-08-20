# PubSub CLI is the missing client for GCP PubSub
The PubSub CLI allows you to interact with PubSub on GCP or an emulator. It is used to manage topics and subscriptions and to publish and receive messages from queues from the command line or scripts.

It does things the client packaged with Google Cloud CLI can't.

## Example usage
Below are some common patterns for using PubSub CLI.

### Receive a specific number of messages
You may want to receive a specific number of messages and you're willing to wait as long as necessary to receive them.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --count 3
```

### Receive a specific number of messages, but we're in a hurry
You may want to receive a specific number of messages but you can't wait around forever.
In this case, we can use the `--count` flag to tell `pubsub` to wait no more than some maximum duration of time since the last message was received.
(Note that this is not an absolute deadline, it's a wait deadline.)

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --count 3 --wait 5s
```

### Assert that we recieve a specific number of messages
You may want to receive a specific number of messages and failing to receive exactly that number of messages
is an error. In this case you can use the `--expect` flag, which is similar to `--count` except that receiving
any other number of messages in the wait duration exits with an error code.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --expect 3 --wait 5s
```
