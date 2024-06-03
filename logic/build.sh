#!/usr/bin/env bash

# build binary
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap seatchecker.go

# package binary
zip seatChecker.zip bootstrap

# cleanup binary
rm bootstrap

mv seatChecker.zip ../out/seatChecker.zip
