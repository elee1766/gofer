package storage

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONStringArray is a custom type for handling JSON arrays stored as strings in the database
type JSONStringArray []string

// Scan implements the sql.Scanner interface for JSONStringArray
func (j *JSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}

	switch v := value.(type) {
	case string:
		if v == "" || v == "[]" {
			*j = []string{}
			return nil
		}
		return json.Unmarshal([]byte(v), j)
	case []byte:
		if len(v) == 0 || string(v) == "[]" {
			*j = []string{}
			return nil
		}
		return json.Unmarshal(v, j)
	default:
		return fmt.Errorf("cannot scan type %T into JSONStringArray", value)
	}
}

// Value implements the driver.Valuer interface for JSONStringArray
func (j JSONStringArray) Value() (driver.Value, error) {
	if j == nil || len(j) == 0 {
		return "[]", nil
	}
	return json.Marshal(j)
}