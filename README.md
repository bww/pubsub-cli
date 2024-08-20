# PubSub CLI is the missing client for GCP PubSub
The PubSub CLI allows you to interact with PubSub on GCP or an emulator. It is used to manage topics and subscriptions and to publish and receive messages from queues from the command line or scripts.

It does things the client packaged with Google Cloud CLI can't.

## Example usage
Below are some common patterns for using PubSub CLI.

### Receive messages until interrupted
In the simplest case, we can just receive messages forever, until we are interrupted.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription
```

### Receive messages until we have waited some duration without receiving one
We may want to receive messages until there are no more immediately available.
In this case we can receive messages until it takes longer than some duration for a message to be available.
On a busy subscription that may never happen and we may effectively receive messages forever.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --wait 5s
```

### Receive a specific number of messages
We may want to receive a specific number of messages and we're willing to wait as long as necessary to receive them.
In this case we can specify the number of messages we want with `--count` and we will wait until we have received
that many messages.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --count 3
```

### Receive a specific number of messages, but we're in a hurry
We may want to receive a specific number of messages but we can't wait around forever.
In this case, we can use the `--wait` flag to tell `pubsub` to wait no more than some maximum duration of time since
the last message was received.

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --count 3 --wait 5s
```

### Assert that we recieve a specific number of messages
We may want to receive a specific number of messages and failing to receive exactly that number of messages
is an error. In this case we can use the `--expect` flag, which is similar to `--count` except that receiving
any other number of messages in the wait duration exits with an error code. (Using `--expect` without `--wait`
is not really useful, although it is allowed.)

```sh
$ pubsub receive data --project some-gcp-project --subscription some-pubsub-subscription --expect 3 --wait 5s
```
