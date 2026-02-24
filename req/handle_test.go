package req

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Example: Add (query extraction + JSON response) ---

func TestAdd(t *testing.T) {
	handler := Handle(func(req *Req, in struct {
		A float64 `query:"a"`
		B float64 `query:"b"`
	}) error {
		return req.JSON(map[string]float64{"sum": in.A + in.B})
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/add?a=1.5&b=2.5", nil)
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var result map[string]float64
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["sum"] != 4 {
		t.Fatalf("sum = %v, want 4", result["sum"])
	}
}

func TestAddMissingParam(t *testing.T) {
	var fieldErrors map[string]string
	handler := Handle(func(req *Req, in struct {
		A float64 `query:"a"`
		B float64 `query:"b"`
	}) error {
		fieldErrors = req.FieldErrors
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/add?a=1", nil)
	handler.ServeHTTP(w, r)

	if msg, ok := fieldErrors["b"]; !ok || msg != `query "b" is required` {
		t.Fatalf(`FieldErrors["b"] = %q, want %q`, msg, `query "b" is required`)
	}
}

// --- Example: CheckField with NotBlank and Between ---

func TestCheckFieldNotBlank(t *testing.T) {
	handler := Handle(func(req *Req, in struct {
		Name string `query:"name"`
	}) error {
		req.CheckField(NotBlank(in.Name), "name", "is required")
		if req.HasErrors() {
			return req.Text(req.FieldErrors["name"])
		}
		return req.Text("ok")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?name=+", nil) // blank (whitespace only)
	handler.ServeHTTP(w, r)
	if body := w.Body.String(); body != "is required" {
		t.Fatalf("body = %q, want %q", body, "is required")
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/?name=alice", nil)
	handler.ServeHTTP(w, r)
	if body := w.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestCheckFieldBetween(t *testing.T) {
	handler := Handle(func(req *Req, in struct {
		Age int `query:"age"`
	}) error {
		req.CheckField(Between(in.Age, 18, 120), "age", "must be between 18 and 120")
		if req.HasErrors() {
			return req.Text(req.FieldErrors["age"])
		}
		return req.Text("ok")
	})

	for _, tc := range []struct {
		query string
		want  string
	}{
		{"?age=17", "must be between 18 and 120"},
		{"?age=18", "ok"},
		{"?age=120", "ok"},
		{"?age=121", "must be between 18 and 120"},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/"+tc.query, nil)
		handler.ServeHTTP(w, r)
		if body := w.Body.String(); body != tc.want {
			t.Errorf("age %s: body = %q, want %q", tc.query, body, tc.want)
		}
	}
}

// --- Example: Custom extractor via WithExtractors + NewExtractor ---

func TestCustomExtractor(t *testing.T) {
	// Simulate a "session" extractor that reads from a custom header.
	extractSession := func(r *http.Request, name string) (string, bool) {
		v := r.Header.Get("X-Session-" + name)
		return v, v != ""
	}

	handler := Handle(func(req *Req, in struct {
		UserID string `session:"user-id"`
	}) error {
		return req.Text(in.UserID)
	}, WithExtractors(
		NewExtractor("session", extractSession),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/dashboard", nil)
	r.Header.Set("X-Session-user-id", "u123")
	handler.ServeHTTP(w, r)

	if body := w.Body.String(); body != "u123" {
		t.Fatalf("body = %q, want %q", body, "u123")
	}
}

// --- Example: Signup (validate tags + Check/CheckField + HasErrors + HTML + Redirect) ---

func signupHandler() http.HandlerFunc {
	return Handle(func(req *Req, in struct {
		Email    string `query:"email" validate:"email"`
		Password string `query:"password"`
		Confirm  string `query:"confirm"`
	}) error {
		req.CheckField(len(in.Password) >= 8, "password", "must be at least 8 characters")
		req.Check(in.Password == in.Confirm, "passwords don't match")
		if req.HasErrors() {
			// Render errors as HTML (simplified for test).
			var parts []string
			for _, e := range req.Errors {
				parts = append(parts, e)
			}
			for f, e := range req.FieldErrors {
				parts = append(parts, f+": "+e)
			}
			return req.HTML(strings.Join(parts, "; "))
		}
		return req.Redirect("/welcome")
	})
}

func TestSignupInvalidEmail(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/signup?email=bad&password=longpassword&confirm=longpassword", nil)
	signupHandler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "email: must be a valid email") {
		t.Fatalf("body = %q, want email error", body)
	}
}

func TestSignupShortPassword(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/signup?email=a@b.com&password=short&confirm=short", nil)
	signupHandler().ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "password: must be at least 8 characters") {
		t.Fatalf("body = %q, want password error", body)
	}
}

func TestSignupPasswordMismatch(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/signup?email=a@b.com&password=longpassword&confirm=different", nil)
	signupHandler().ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "passwords don't match") {
		t.Fatalf("body = %q, want mismatch error", body)
	}
}

func TestSignupSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/signup?email=a@b.com&password=longpassword&confirm=longpassword", nil)
	signupHandler().ServeHTTP(w, r)

	if w.Code != 302 {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/welcome" {
		t.Fatalf("Location = %q, want /welcome", loc)
	}
}

// --- Example: Strict adapter ---

func Strict[T any](fn func(*Req, T) error) http.HandlerFunc {
	return Handle(func(req *Req, in T) error {
		if req.HasErrors() {
			return HTTPError(400, req.Error())
		}
		if err := fn(req, in); err != nil {
			return err
		}
		if req.HasErrors() {
			return HTTPError(400, req.Error())
		}
		return nil
	})
}

func TestStrictRejectsDecodeErrors(t *testing.T) {
	handler := Strict(func(req *Req, in struct {
		N int `query:"n"`
	}) error {
		return req.Text("ok")
	})

	// Missing required field → auto 400.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); !strings.Contains(body, "is required") {
		t.Fatalf("body = %q, want 'is required'", body)
	}
}

func TestStrictRejectsHandlerErrors(t *testing.T) {
	handler := Strict(func(req *Req, in struct {
		A int `query:"a"`
		B int `query:"b"`
	}) error {
		req.Check(in.B != 0, "B must not be zero")
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?a=1&b=0", nil)
	handler.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); !strings.Contains(body, "B must not be zero") {
		t.Fatalf("body = %q, want 'B must not be zero'", body)
	}
}

func TestStrictSuccess(t *testing.T) {
	handler := Strict(func(req *Req, in struct {
		N int `query:"n"`
	}) error {
		return req.Text("ok")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?n=42", nil)
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

// --- Pointer fields: missing (nil) vs. present-but-empty (non-nil) ---

func TestPointerFieldMissing(t *testing.T) {
	var got *string
	handler := Handle(func(req *Req, in struct {
		Name *string `query:"name"`
	}) error {
		got = in.Name
		return req.NoContent()
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if got != nil {
		t.Fatalf("Name = %q, want nil", *got)
	}
}

func TestPointerFieldPresent(t *testing.T) {
	var got *string
	handler := Handle(func(req *Req, in struct {
		Name *string `query:"name"`
	}) error {
		got = in.Name
		return req.NoContent()
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?name=", nil) // present but empty
	handler.ServeHTTP(w, r)

	if got == nil {
		t.Fatal("Name = nil, want non-nil")
	}
	if *got != "" {
		t.Fatalf("Name = %q, want %q", *got, "")
	}
}

// --- Nested structs: decode recurses into struct-typed fields ---

func TestNestedStruct(t *testing.T) {
	type Address struct {
		City  string `query:"city"`
		State string `query:"state"`
	}

	handler := Handle(func(req *Req, in struct {
		Name    string `query:"name"`
		Address Address
	}) error {
		return req.Text(in.Name + " from " + in.Address.City + ", " + in.Address.State)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?name=Alice&city=Portland&state=OR", nil)
	handler.ServeHTTP(w, r)

	want := "Alice from Portland, OR"
	if body := w.Body.String(); body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}
