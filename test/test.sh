#!/usr/bin/env bash

set -eo pipefail

# where am i?
me_home=$(dirname "$0")
me_home=$(cd "$me_home" && pwd)

# parse arguments
args=$(getopt d $*)
# set arguments
set -- $args
# progess arguments
for i; do
  case "$i"
  in
    -d)
      debug="true"
      shift;;
    --)
      shift;
      break;;
  esac
done

# where is the project
project_home=$(cd "$me_home/.." && pwd)
# the normal command
PUBSUB=${project_home}/target/$(go env GOOS)_$(go env GOARCH)/bin/pubsub
# the debugging command
PUBSUB_DEBUG="dlv debug $project_home/cmd --"

# this is what we're testing
if [ ! -z "$debug" ]; then
  PUBSUB=$PUBSUB_DEBUG
fi

export PUBSUB_PROJECT=pubsub
export PUBSUB_TOPIC=pubsub-integrate
export PUBSUB_SUBSCRIPTION=pubsub-integrate-subscription

# create the topic and subscription we'll use
$PUBSUB topic new $PUBSUB_TOPIC
$PUBSUB subscription new --topic $PUBSUB_TOPIC $PUBSUB_SUBSCRIPTION

# drain any previously-existing messages
$PUBSUB receive data --subscription $PUBSUB_SUBSCRIPTION --wait 3s --verbose

# publish a messgae
$PUBSUB publish data --topic $PUBSUB_TOPIC "$me_home/data.txt" --quiet
$PUBSUB receive data --subscription $PUBSUB_SUBSCRIPTION --expect 1 --wait 1s
$PUBSUB receive data --subscription $PUBSUB_SUBSCRIPTION --expect 1 --wait 1s || echo "None, as expected"
$PUBSUB receive data --subscription $PUBSUB_SUBSCRIPTION --expect 0 --wait 1s || echo "None, as expected"
