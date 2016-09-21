#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o cf-predix-analytics-plugin.linux64 *.go
GOOS=linux GOARCH=386 go build -o cf-predix-analytics-plugin.linux32 *.go
GOOS=windows GOARCH=amd64 go build -o cf-predix-analytics-plugin.win64 *.go 
GOOS=windows GOARCH=386 go build -o cf-predix-analytics-plugin.win32 *.go
GOOS=darwin GOARCH=amd64 go build -o cf-predix-analytics-plugin.osx *.go
