# Handle

This package provides a couple of convenient HTTP handlers as well as a mechanism
for simpler handler definitions that include extracting input strings from the URL
path, query params, cookies, and headers. The system is extensible, so eg. an app
can define special extractors for extracting from session state or Datastar signals.
See `TestCustomExtractor` for an example of a session extractor.

---

`Handle` is a generic adapter for net/http handlers that separates core handler
logic from the repetitive work of input parsing, error reporting, and response
formatting.

A handler is a plain function, where Req wraps a response writer and request
for handler convenience:

```go
func(*Req, T) error
```

`T` is a data struct whose fields declare where each input comes from via
struct tags — path, query, cookie, or header. When multiple source
tags are present, extraction priority follows the extractor list order.
Non-pointer fields are required: if no extractor finds a value, decode
reports "is required". Pointer fields are optional: missing values stay nil.
Fields may also carry validate tags for declarative validation
(notblank, email, min=N, max=N):

```go
mux.HandleFunc("GET /add", Handle(func(req *Req, in struct {
    A float64 `query:"a"`
    B float64 `query:"b"`
}) error {
    return req.JSON(map[string]float64{"sum": in.A + in.B})
}))
```

Boolean helper functions (NotBlank, IsEmail, Between, In, etc.) are
available for composing custom validations with Check and CheckField:

```go
req.CheckField(NotBlank(in.Name), "Name", "is required")
req.CheckField(Between(in.Age, 18, 120), "Age", "must be between 18 and 120")
```

Options can be passed to append additional extractors and validators:

```go
mux.HandleFunc("GET /dashboard", Handle(handleDashboard,
    WithExtractors(
        NewExtractor("session", extractSession),
    ),
))
```

Handle always calls the handler. Decode errors and validate-tag errors are
pre-populated into req.FieldErrors, letting the handler add more via
Check/CheckField and decide how to respond (e.g. re-render a form with
inline errors):

```go
mux.HandleFunc("POST /signup", Handle(func(req *Req, in struct {
    Email    string `query:"email" validate:"email"`
    Password string `query:"password"`
    Confirm  string `query:"confirm"`
}) error {
    req.CheckField(len(in.Password) >= 8, "Password", "must be at least 8 characters")
    req.Check(in.Password == in.Confirm, "passwords don't match")
    if req.HasErrors() {
        return req.HTML(renderForm(req.Errors, req.FieldErrors))
    }
    createUser(in.Email, in.Password)
    return req.Redirect("/welcome")
}))
```

For a strict adapter that auto-400s on any errors, wrap Handle:

```go
func Strict[T any](fn func(*Req, T) error) http.HandlerFunc {
    return Handle(func(req *Req, in T) error {
        if req.HasErrors() {
            return HTTPError(400, req.ErrorsMessage())
        }
        if err := fn(req, in); err != nil {
            return err
        }
        if req.HasErrors() {
            return HTTPError(400, req.ErrorsMessage())
        }
        return nil
    })
}
```

Pointer fields distinguish "missing" (nil) from "present but empty" (non-nil).
Decode recurses into struct-typed fields that have no extraction tags, so
nested structs without source tags are populated from their own fields' tags.

Body parsing (JSON, form data, etc.) is intentionally left to handlers.
Extractors return (string, bool), so structured data doesn't fit the pattern, and
the implicit "is required" check can't see inside a decoded struct — after
json.Unmarshal, a missing key is indistinguishable from a zero value.
Deserialization also has too many knobs (unknown fields, number precision,
custom decoders) for a single struct tag to capture. Use json.NewDecoder in your
handler and call CheckField or req.Validate(&myStruct) to validate the result.
