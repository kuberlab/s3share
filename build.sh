#!/usr/bin/env bash

env GOOS=linux go build -v -ldflags="-s -w" share.go
