package req

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/starfederation/datastar-go/datastar"
)

type signalsKeyType struct{}

var signalsKey signalsKeyType

// withSignals reads DataStar signals from the request and stores
// the resulting map in the request context for extractSignal to use.
// If ReadSignals fails (e.g. malformed JSON body or missing query param),
// the error is silently ignored and individual signal fields will appear
// as missing, triggering the normal "is required" field-level errors.
func withSignals(r *http.Request) *http.Request {
	signals := map[string]any{}
	// Error ignored: a failed read leaves signals empty, so fields
	// with signal tags will report as missing via normal validation.
	_ = datastar.ReadSignals(r, &signals)
	return r.WithContext(context.WithValue(r.Context(), signalsKey, signals))
}

func extractSignal(r *http.Request, name string) (string, bool) {
	signals, _ := r.Context().Value(signalsKey).(map[string]any)
	v, found := signals[name]
	if !found {
		return "", false
	}
	return fmt.Sprint(v), true
}

// hasSignalTag reports whether t or any nested struct field has a "signal" tag.
func hasSignalTag(t reflect.Type) bool {
	for i := range t.NumField() {
		f := t.Field(i)
		if _, ok := f.Tag.Lookup("signal"); ok {
			return true
		}
		if f.Type.Kind() == reflect.Struct {
			if hasSignalTag(f.Type) {
				return true
			}
		}
	}
	return false
}
