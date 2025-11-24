package toolkit

import "encoding/json"

// ToJSON serializes the given interface to JSON bytes
func ToJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func FromJSON[T any](b []byte) (T, error) {
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}
