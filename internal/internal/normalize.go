package internal

import (
	"fmt"
	"time"
)

// NormalizeSetting checks and converts the newValue to the appropriate type
func NormalizeSetting(currentValue interface{}, newValue interface{}) (interface{}, error) {
	switch currentValue.(type) {
	case string:
		if strValue, ok := newValue.(string); ok {
			return strValue, nil
		}
	case int:
		if intValue, ok := newValue.(int); ok {
			return intValue, nil
		}
	case float64:
		if floatValue, ok := newValue.(float64); ok {
			return floatValue, nil
		}
	case bool:
		if boolValue, ok := newValue.(bool); ok {
			return boolValue, nil
		}
	case time.Duration:
		switch v := newValue.(type) {
		case time.Duration:
			return v, nil
		case string:
			parsedDuration, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("invalid duration format: %v", err)
			}
			return parsedDuration, nil
		case float64:
			return time.Duration(v * float64(time.Second)), nil
		default:
			return nil, fmt.Errorf("invalid data type for duration: expected string, float64, or time.Duration")
		}
	case []interface{}:
		if arrayValue, ok := newValue.([]interface{}); ok {
			return arrayValue, nil
		}
	case map[string]interface{}:
		if mapValue, ok := newValue.(map[string]interface{}); ok {
			return mapValue, nil
		}
	}
	return nil, fmt.Errorf("type mismatch: cannot set %T to %T", newValue, currentValue)
}
