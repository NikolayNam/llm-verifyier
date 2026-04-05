package main

import "time"

func defaultRunID() string {
	return time.Now().Local().Format("20060102T150405-0700")
}
