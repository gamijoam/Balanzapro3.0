package domain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// DecodeValue decodes stored JSON using the declared type, never float64.
// Null is not a configuration value.
func (r Registry) DecodeValue(key string, raw []byte) (any, error) {
	k, err := r.key(key)
	if err != nil {
		return nil, err
	}
	invalid := fmt.Errorf("%w: invalid stored value for %s", ErrInvalidValue, key)
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, invalid
	}
	var value any
	switch k.Type {
	case TypeBool:
		var v bool
		if json.Unmarshal(raw, &v) != nil {
			return nil, invalid
		}
		value = v
	case TypeInteger:
		var v int64
		if json.Unmarshal(raw, &v) != nil {
			return nil, invalid
		}
		value = v
	case TypeDecimal:
		var v qty.Q
		if json.Unmarshal(raw, &v) != nil {
			return nil, invalid
		}
		value = v
	case TypeString, TypeEnum:
		var v string
		if json.Unmarshal(raw, &v) != nil {
			return nil, invalid
		}
		value = v
	default:
		return nil, invalid
	}
	return r.Validate(key, value)
}
