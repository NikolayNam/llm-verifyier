package main

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if _, err := loadBenchmarkPromptProfiles(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "loadBenchmarkPromptProfiles() error: %v\n", err)
		os.Exit(1)
	}
	if _, err := loadNDBenchmarkPromptProfiles(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "loadNDBenchmarkPromptProfiles() error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
