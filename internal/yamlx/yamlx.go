package yamlx

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Decode unmarshals YAML and rejects unknown fields.
func Decode(data []byte, v interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("yaml: %w", err)
	}
	return nil
}
