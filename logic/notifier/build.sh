#!/usr/bin/env bash

# build binary
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap notifier.go

# package binary
zip notifier.zip bootstrap

# cleanup binary
rm bootstrap

mv notifier.zip ../../out/notifier.zip
