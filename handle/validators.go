package handle

import (
	"cmp"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// --- Boolean helper functions ---

// NotBlank reports whether value contains non-whitespace characters.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

func NonZero[T comparable](value T) bool {
	var zero T
	return value != zero
}

var rxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// IsEmail reports whether value looks like a valid email address.
func IsEmail(value string) bool {
	return rxEmail.MatchString(value)
}

// MinRunes reports whether value has at least n runes.
func MinRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// MaxRunes reports whether value has at most n runes.
func MaxRunes(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// Between reports whether value is in the range [min, max].
func Between[T cmp.Ordered](value, min, max T) bool {
	return value >= min && value <= max
}

// Matches reports whether value matches the regular expression rx.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// In reports whether value is in the safelist.
func In[T comparable](value T, safelist ...T) bool {
	return slices.Contains(safelist, value)
}

// AllIn reports whether every element of values is in the safelist.
func AllIn[T comparable](values []T, safelist ...T) bool {
	for _, v := range values {
		if !In(v, safelist...) {
			return false
		}
	}
	return true
}

// NotIn reports whether value is not in the blocklist.
func NotIn[T comparable](value T, blocklist ...T) bool {
	return !In(value, blocklist...)
}

// NoDuplicates reports whether all elements of values are unique.
func NoDuplicates[T comparable](values []T) bool {
	seen := make(map[T]struct{}, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

// IsURL reports whether value is a valid URL with a scheme and host.
func IsURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// --- Struct tag validation ---

// validator validates a field value and returns "" if valid or an error message.
// arg is the part after "=" in a validate tag (e.g. "18" in "min=18"; empty for
// rules like "email").
type validator struct {
	name     string
	validate func(fieldValue any, arg string) string
}

// NewValidator adapts a boolean helper with no tag argument to a struct tag validator.
func NewValidator[V any](name, msg string, fn func(V) bool) validator {
	return validator{name: name, validate: func(fieldValue any, _ string) string {
		v, ok := fieldValue.(V)
		if !ok {
			panic(fmt.Sprintf("validate: rule %q expected field type %T, got %T", name, v, fieldValue))
		}
		if !fn(v) {
			return msg
		}
		return ""
	}}
}

// Parseable is the set of types that can be parsed from a struct tag argument.
type Parseable interface {
	~int | ~float64 | ~string
}

func parseTag[T Parseable](s string) T {
	var zero T
	switch any(zero).(type) {
	case int:
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("validate: malformed tag arg %q: %v", s, err))
		}
		return any(n).(T)
	case float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			panic(fmt.Sprintf("validate: malformed tag arg %q: %v", s, err))
		}
		return any(n).(T)
	case string:
		return any(s).(T)
	}
	panic("unreachable")
}

// NewValidatorWithArg adapts a boolean helper with one parsed tag argument to a struct tag validator.
// msg may contain one %s verb for the tag argument value.
func NewValidatorWithArg[V any, A Parseable](name, msg string, fn func(V, A) bool) validator {
	return validator{name: name, validate: func(fieldValue any, arg string) string {
		v, ok := fieldValue.(V)
		if !ok {
			panic(fmt.Sprintf("validate: rule %q expected field type %T, got %T", name, v, fieldValue))
		}
		if !fn(v, parseTag[A](arg)) {
			return fmt.Sprintf(msg, arg)
		}
		return ""
	}}
}

func minInt(v int, min int) bool       { return v >= min }
func maxInt(v int, max int) bool       { return v <= max }
func minFloat(v float64, min float64) bool { return v >= min }
func maxFloat(v float64, max float64) bool { return v <= max }

func numericValidator(name, msg string, intFn func(int, int) bool, floatFn func(float64, float64) bool) validator {
	return validator{name: name, validate: func(fieldValue any, arg string) string {
		switch v := fieldValue.(type) {
		case int:
			if !intFn(v, parseTag[int](arg)) {
				return fmt.Sprintf(msg, arg)
			}
		case float64:
			if !floatFn(v, parseTag[float64](arg)) {
				return fmt.Sprintf(msg, arg)
			}
		default:
			panic(fmt.Sprintf("validate: rule %q expected numeric type, got %T", name, fieldValue))
		}
		return ""
	}}
}

var defaultValidators = []validator{
	NewValidator("notblank", "cannot be blank", NotBlank),
	NewValidator("email", "must be a valid email", IsEmail),
	numericValidator("min", "must be at least %s", minInt, minFloat),
	numericValidator("max", "must be at most %s", maxInt, maxFloat),
}
