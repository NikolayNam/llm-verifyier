package planner

import "time"

func defaultTimestampRunID() string {
	return time.Now().Local().Format("20060102T150405-0700")
}
