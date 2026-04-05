package naturaldeduction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func DecodeJSON(data []byte) (*Proof, error) {
	var proof Proof

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&proof); err != nil {
		return nil, NewError(ErrorClassSchema, fmt.Errorf("decode natural deduction json: %w", err))
	}
	if dec.More() {
		return nil, NewError(ErrorClassSchema, fmt.Errorf("decode natural deduction json: trailing data after top-level object"))
	}
	if err := proof.Validate(); err != nil {
		return nil, NewError(classifyValidationError(err), fmt.Errorf("validate natural deduction json: %w", err))
	}

	return &proof, nil
}

func LoadJSONFile(path string) (*Proof, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewError(ErrorClassInternal, fmt.Errorf("read natural deduction file: %w", err))
	}
	return DecodeJSON(data)
}

func classifyValidationError(err error) ErrorClass {
	if err == nil {
		return ""
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "parse formula:"):
		return ErrorClassParse
	case strings.Contains(message, " is required"),
		strings.Contains(message, " must be "),
		strings.Contains(message, "must not define"),
		strings.Contains(message, "must not be"),
		strings.Contains(message, "expected "),
		strings.Contains(message, "unsupported step kind"),
		strings.Contains(message, "scope must be positive"),
		strings.Contains(message, "must omit scope"),
		strings.Contains(message, "must declare the current scope"):
		return ErrorClassSchema
	default:
		return ErrorClassValidation
	}
}
