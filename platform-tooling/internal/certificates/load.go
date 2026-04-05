package certificates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func DecodeJSON(data []byte) (*Certificate, error) {
	var cert Certificate

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cert); err != nil {
		return nil, NewError(ErrorClassSchema, fmt.Errorf("decode certificate json: %w", err))
	}
	if dec.More() {
		return nil, NewError(ErrorClassSchema, fmt.Errorf("decode certificate json: trailing data after top-level object"))
	}
	if err := cert.Validate(); err != nil {
		return nil, NewError(ErrorClassSchema, fmt.Errorf("validate certificate json: %w", err))
	}

	return &cert, nil
}

func LoadJSONFile(path string) (*Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewError(ErrorClassInternal, fmt.Errorf("read certificate file: %w", err))
	}
	return DecodeJSON(data)
}
