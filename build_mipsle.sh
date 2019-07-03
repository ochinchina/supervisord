#!/bin/sh

GOPROXY=https://goproxy.io GOOS=linux GOARCH=mipsle go build -ldflags "-s -w"
