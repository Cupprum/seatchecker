#!/usr/bin/env bash

# build binary
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap seatchecker.go

# package binary
zip seatchecker.zip bootstrap

# cleanup binary
rm bootstrap

mv seatchecker.zip ../out/seatchecker.zip
