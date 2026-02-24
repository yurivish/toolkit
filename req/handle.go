package req

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// --- Decoder ---

// Decoder extracts struct fields from an HTTP request using struct tags.
type Decoder struct {
	extractors []extractor
}

// DecodeResult holds the outcome of a Decode call.
// FieldErrors keys are the external tag values (first matching extractor tag),
// not Go field names.
type DecodeResult struct {
	FieldErrors map[string]string // per-field parse failures
}

// Decode populates dst (must be *struct) from the request using struct tags.
func (d *Decoder) Decode(r *http.Request, dst any) (DecodeResult, error) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return DecodeResult{}, fmt.Errorf("decode: dst must be *struct")
	}
	result := DecodeResult{
		FieldErrors: make(map[string]string),
	}
	decodeStruct(r, v.Elem(), d.extractors, result.FieldErrors)
	return result, nil
}

// externalName returns the first tag value found on a field from the
// extractor list, falling back to f.Name. This is used as the key in
// FieldErrors so that error keys match the external names callers use.
func externalName(f reflect.StructField, extractors []extractor) string {
	for _, ex := range extractors {
		if tag, ok := f.Tag.Lookup(ex.tag); ok {
			return tag
		}
	}
	return f.Name
}

// requiredMessage builds a descriptive "is required" message listing all
// sources tried. Single source → `query "name" is required`.
// Multiple → `query "name" or header "x-name" is required`.
func requiredMessage(f reflect.StructField, extractors []extractor) string {
	var parts []string
	for _, ex := range extractors {
		if tag, ok := f.Tag.Lookup(ex.tag); ok {
			parts = append(parts, fmt.Sprintf("%s %q", ex.tag, tag))
		}
	}
	if len(parts) == 0 {
		return "is required"
	}
	return strings.Join(parts, " or ") + " is required"
}

// decodeStruct tries each extractor in order via f.Tag.Lookup. If no tag
// produced a value and the field is a struct, it recurses. Non-pointer fields
// with extraction tags that aren't matched get an "is required" error.
func decodeStruct(r *http.Request, v reflect.Value, extractors []extractor, errs map[string]string) {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		fv := v.Field(i)
		matched := false

		// Try each extractor in order.
		for _, ex := range extractors {
			if tag, ok := f.Tag.Lookup(ex.tag); ok {
				val, found := ex.extract(r, tag)
				if found {
					if err := decodeField(fv, val); err != nil {
						key := externalName(f, extractors)
						if _, exists := errs[key]; !exists {
							errs[key] = fmt.Sprintf("%s %q: %s", ex.tag, tag, err)
						}
					}
					matched = true
					break
				}
			}
		}

		if !matched && fv.Kind() == reflect.Struct {
			decodeStruct(r, fv, extractors, errs)
		} else if !matched && fv.Kind() != reflect.Pointer {
			errs[externalName(f, extractors)] = requiredMessage(f, extractors)
		}
	}
}

func decodeField(fv reflect.Value, s string) error {
	// If a pointer, create a new scalar before decoding into it.
	if fv.Kind() == reflect.Pointer {
		fv.Set(reflect.New(fv.Type().Elem()))
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type %s", fv.Kind())
	}
	return nil
}

// --- Validator ---

// Validator collects non-field errors and per-field errors into a single place.
// FieldErrors keys are the external tag values (first matching extractor tag),
// not Go field names. For example, a field `Name string `query:"name"“ uses
// "name" as the key.
type Validator struct {
	Errors      []string          // non-field errors ("passwords don't match")
	FieldErrors map[string]string // field -> error message; first error per field wins
	extractors  []extractor
	validators  []validator
}

// AddError appends a non-field error.
func (v *Validator) AddError(message string) {
	v.Errors = append(v.Errors, message)
}

// AddFieldError records an error for field. The first error per field wins;
// subsequent errors for the same field are ignored.
func (v *Validator) AddFieldError(field, message string) {
	if _, exists := v.FieldErrors[field]; !exists {
		v.FieldErrors[field] = message
	}
}

// Check adds a non-field error if ok is false.
func (v *Validator) Check(ok bool, message string) {
	if !ok {
		v.AddError(message)
	}
}

// CheckField records an error for field if ok is false.
// The first error per field wins; subsequent errors for the same field are ignored.
func (v *Validator) CheckField(ok bool, field, message string) {
	if !ok {
		v.AddFieldError(field, message)
	}
}

// HasErrors reports whether any errors have been recorded.
func (v *Validator) HasErrors() bool {
	return len(v.Errors) > 0 || len(v.FieldErrors) > 0
}

// ErrorsMessage formats all errors as a single string.
func (v *Validator) ErrorsMessage() string {
	msgs := make([]string, 0, len(v.Errors)+len(v.FieldErrors))
	msgs = append(msgs, v.Errors...)
	for field, msg := range v.FieldErrors {
		msgs = append(msgs, field+": "+msg)
	}
	slices.Sort(msgs)
	return strings.Join(msgs, "; ")
}

// validateStruct walks struct fields, reads "validate" tags, and writes errors
// into FieldErrors. It skips fields that already have an error (first-error-per-field
// wins). It recurses into struct-typed fields for nested validation.
func (v *Validator) validateStruct(rv reflect.Value) {
	t := rv.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		fv := rv.Field(i)

		if fv.Kind() == reflect.Struct {
			v.validateStruct(fv)
			continue
		}

		tag, ok := f.Tag.Lookup("validate")
		if !ok {
			continue
		}
		key := externalName(f, v.extractors)
		if _, exists := v.FieldErrors[key]; exists {
			continue
		}

	fieldValidators:
		for rule := range strings.SplitSeq(tag, ",") {
			name, arg, _ := strings.Cut(rule, "=")
			for _, vr := range v.validators {
				if vr.name == name {
					if msg := vr.validate(fv.Interface(), arg); msg != "" {
						v.FieldErrors[key] = msg
						break fieldValidators
					}
					break
				}
			}
		}
	}
}

// --- Req ---

// Req wraps a response writer and request for handler convenience.
type Req struct {
	Validator
	W http.ResponseWriter
	R *http.Request
}

// Validate runs struct-tag validation (email, min, max, etc.) on dst,
// writing any errors into req.FieldErrors. Useful after manually decoding
// a JSON body into a struct with validate tags.
func (req *Req) Validate(dst any) {
	v := reflect.ValueOf(dst)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	req.validateStruct(v)
}

// --- Response helpers ---

// JSON writes v as JSON with status 200.
func (req *Req) JSON(v any) error {
	return req.JSONStatus(200, v)
}

// JSONStatus writes v as JSON with the given status code.
func (req *Req) JSONStatus(status int, v any) error {
	req.W.Header().Set("Content-Type", "application/json")
	req.W.WriteHeader(status)
	return json.NewEncoder(req.W).Encode(v)
}

// HTML writes s as text/html with status 200.
func (req *Req) HTML(s string) error {
	req.W.Header().Set("Content-Type", "text/html")
	_, err := fmt.Fprint(req.W, s)
	return err
}

// Text writes s as text/plain with status 200.
func (req *Req) Text(s string) error {
	req.W.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprint(req.W, s)
	return err
}

// Redirect sends a 302 redirect to the given URL.
func (req *Req) Redirect(url string) error {
	http.Redirect(req.W, req.R, url, http.StatusFound)
	return nil
}

// NoContent sends a 204 No Content response.
func (req *Req) NoContent() error {
	req.W.WriteHeader(http.StatusNoContent)
	return nil
}

// --- Error types ---

// httpError is a sentinel error carrying an HTTP status code and message.
type httpError struct {
	status int
	msg    string
}

func (e httpError) Error() string { return e.msg }

// HTTPError returns an error that the Handle adapter unwraps into an HTTP error response.
func HTTPError(status int, msg string) error {
	return httpError{status: status, msg: msg}
}

// --- Adapters ---

type handleOption func(*handleConfig)

type handleConfig struct {
	extractors []extractor
	validators []validator
}

// WithExtractors returns a handle option that appends additional extractors.
func WithExtractors(extractors ...extractor) handleOption {
	return func(c *handleConfig) { c.extractors = append(c.extractors, extractors...) }
}

// WithValidators returns a handle option that appends additional validators.
func WithValidators(validators ...validator) handleOption {
	return func(c *handleConfig) { c.validators = append(c.validators, validators...) }
}

// Handle is a generic adapter that decodes request input into T,
// then always calls fn. Per-field decode errors are written into Req.FieldErrors
// so the handler can inspect them alongside its own Check/CheckField calls.
func Handle[T any](fn func(*Req, T) error, opts ...handleOption) http.HandlerFunc {
	extractors, validators := defaultExtractors, defaultValidators
	if len(opts) > 0 {
		cfg := handleConfig{extractors: slices.Clone(extractors), validators: slices.Clone(validators)}
		for _, opt := range opts {
			opt(&cfg)
		}
		extractors, validators = cfg.extractors, cfg.validators
	}
	decoder := Decoder{extractors: extractors}
	return func(w http.ResponseWriter, r *http.Request) {
		var input T
		result, err := decoder.Decode(r, &input)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		req := &Req{
			W:         w,
			R:         r,
			Validator: Validator{FieldErrors: result.FieldErrors, extractors: extractors, validators: validators},
		}
		req.validateStruct(reflect.ValueOf(&input).Elem())
		if err := fn(req, input); err != nil {
			if he, ok := errors.AsType[httpError](err); ok {
				http.Error(w, he.msg, he.status)
			} else {
				http.Error(w, err.Error(), 500)
			}
		}
	}
}
