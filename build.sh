#!/bin/bash

env GOOS=linux GOARCH=amd64 go build -o ./builds/linux/psminimize
env GOOS=windows GOARCH=amd64 go build -o ./builds/windows/psminimize