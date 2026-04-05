package config

import (
	"fmt"
	"strings"
)

const (
	InterruptPolicyKeep            = "keep"
	InterruptPolicyDropIfNoResults = "drop_if_no_results"
	InterruptPolicyDropAlways      = "drop_always"

	DefaultInterruptPolicy = InterruptPolicyDropIfNoResults
)

func NormalizeInterruptPolicy(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return DefaultInterruptPolicy
	}
	return value
}

func ValidateInterruptPolicy(value string) error {
	switch NormalizeInterruptPolicy(value) {
	case InterruptPolicyKeep, InterruptPolicyDropIfNoResults, InterruptPolicyDropAlways:
		return nil
	default:
		return fmt.Errorf("interrupt_policy must be one of keep|drop_if_no_results|drop_always")
	}
}
