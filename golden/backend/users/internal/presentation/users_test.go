package presentation

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"

	"golden-app/backend/users/internal/application"
	"golden-app/backend/users/internal/infrastructure/memory"
)

const (
	testEmail    = "ada@example.com"
	testPassword = "correct horse battery staple"
)

func newTestAPI(t *testing.T, env string) humatest.TestAPI {
	t.Helper()

	store := memory.NewStore()
	svc := application.NewUserService(
		store,
		store,
		memory.NewSessionIssuer(store.Users(), time.Hour),
		memory.NewNotifier(),
		application.NewArgon2Hasher(),
		env,
	)

	_, api := humatest.New(t)
	RegisterUsers(api, svc, NewCookieOptions(env), slog.New(slog.NewTextHandler(io.Discard, nil)))
	return api
}

func decode(t *testing.T, body []byte, into any) {
	t.Helper()
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("decoding response %s: %v", body, err)
	}
}

// register returns the dev token the non-production response exposes.
func register(t *testing.T, api humatest.TestAPI, email, password string) string {
	t.Helper()

	resp := api.Post("/users/register", map[string]string{"email": email, "password": password})
	if resp.Code != http.StatusAccepted {
		t.Fatalf("register: expected 202, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Token string `json:"token"`
	}
	decode(t, resp.Body.Bytes(), &body)
	if body.Token == "" {
		t.Fatal("expected a dev token outside production")
	}
	return body.Token
}

func TestRegister_ExistingEmailConflicts(t *testing.T) {
	api := newTestAPI(t, "test")

	first := api.Post("/users/register", map[string]string{"email": testEmail, "password": testPassword})
	if first.Code != http.StatusAccepted {
		t.Fatalf("first register: expected 202, got %d: %s", first.Code, first.Body.String())
	}

	second := api.Post("/users/register", map[string]string{"email": testEmail, "password": testPassword})
	if second.Code != http.StatusConflict {
		t.Fatalf("second register: expected 409, got %d: %s", second.Code, second.Body.String())
	}

	var body struct {
		Detail string `json:"detail"`
	}
	decode(t, second.Body.Bytes(), &body)
	if body.Detail != "users: email already registered" {
		t.Fatalf("expected the email-taken detail, got %q", body.Detail)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Post("/users/register", map[string]string{"email": testEmail, "password": "short"})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLogin_SetsSessionCookie(t *testing.T) {
	api := newTestAPI(t, "test")
	token := register(t, api, testEmail, testPassword)

	confirm := api.Post("/users/confirm-email", map[string]string{"token": token})
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm-email: expected 200, got %d: %s", confirm.Code, confirm.Body.String())
	}

	resp := api.Post("/users/login", map[string]string{"email": testEmail, "password": testPassword})
	if resp.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	cookie := sessionCookieFrom(t, resp.Header().Get("Set-Cookie"))
	if cookie.Value == "" {
		t.Fatal("expected a non-empty session cookie value")
	}
	if !cookie.HttpOnly {
		t.Error("expected the session cookie to be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", cookie.SameSite)
	}

	var body struct {
		Email         string   `json:"email"`
		Role          string   `json:"role"`
		Permissions   []string `json:"permissions"`
		EmailVerified bool     `json:"emailVerified"`
	}
	decode(t, resp.Body.Bytes(), &body)
	if body.Email != testEmail || !body.EmailVerified {
		t.Fatalf("unexpected login body: %+v", body)
	}
	if body.Permissions == nil {
		t.Error("expected permissions to serialise as an empty array, not null")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	api := newTestAPI(t, "test")
	register(t, api, testEmail, testPassword)

	resp := api.Post("/users/login", map[string]string{"email": testEmail, "password": "wrong password"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLogin_UnconfirmedEmailInProduction(t *testing.T) {
	api := newTestAPI(t, "prod")

	if resp := api.Post("/users/register", map[string]string{"email": testEmail, "password": testPassword}); resp.Code != http.StatusAccepted {
		t.Fatalf("register: expected 202, got %d", resp.Code)
	}

	resp := api.Post("/users/login", map[string]string{"email": testEmail, "password": testPassword})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestMe_ReadsSessionCookie(t *testing.T) {
	api := newTestAPI(t, "test")
	register(t, api, testEmail, testPassword)

	login := api.Post("/users/login", map[string]string{"email": testEmail, "password": testPassword})
	if login.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", login.Code, login.Body.String())
	}
	cookie := sessionCookieFrom(t, login.Header().Get("Set-Cookie"))

	resp := api.Get("/users/me", "Cookie: "+SessionCookieName+"="+cookie.Value)
	if resp.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Email string `json:"email"`
	}
	decode(t, resp.Body.Bytes(), &body)
	if body.Email != testEmail {
		t.Fatalf("expected %q, got %q", testEmail, body.Email)
	}
}

func TestMe_WithoutSessionCookie(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Get("/users/me")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLogout_ClearsCookieAndRevokesSession(t *testing.T) {
	api := newTestAPI(t, "test")
	register(t, api, testEmail, testPassword)

	login := api.Post("/users/login", map[string]string{"email": testEmail, "password": testPassword})
	cookie := sessionCookieFrom(t, login.Header().Get("Set-Cookie"))
	header := "Cookie: " + SessionCookieName + "=" + cookie.Value

	logout := api.Post("/users/logout", header)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d: %s", logout.Code, logout.Body.String())
	}
	cleared := sessionCookieFrom(t, logout.Header().Get("Set-Cookie"))
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("expected the logout cookie to clear the session, got %+v", cleared)
	}

	if resp := api.Get("/users/me", header); resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected the revoked session to be rejected, got %d", resp.Code)
	}
}

func TestConfirmEmail_InvalidToken(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Post("/users/confirm-email", map[string]string{"token": "not-a-real-token"})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasswordReset_RoundTrip(t *testing.T) {
	api := newTestAPI(t, "test")
	register(t, api, testEmail, testPassword)

	reset := api.Post("/users/password-reset", map[string]string{"email": testEmail})
	if reset.Code != http.StatusAccepted {
		t.Fatalf("password-reset: expected 202, got %d: %s", reset.Code, reset.Body.String())
	}
	var resetBody struct {
		Token string `json:"token"`
	}
	decode(t, reset.Body.Bytes(), &resetBody)
	if resetBody.Token == "" {
		t.Fatal("expected a dev reset token outside production")
	}

	const newPassword = "an entirely new password"
	confirm := api.Post("/users/password-reset/confirm", map[string]string{
		"token":    resetBody.Token,
		"password": newPassword,
	})
	if confirm.Code != http.StatusOK {
		t.Fatalf("password-reset/confirm: expected 200, got %d: %s", confirm.Code, confirm.Body.String())
	}

	if resp := api.Post("/users/login", map[string]string{"email": testEmail, "password": newPassword}); resp.Code != http.StatusOK {
		t.Fatalf("expected the new password to work, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasswordReset_UnknownEmailStillAccepted(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Post("/users/password-reset", map[string]string{"email": "nobody@example.com"})
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasswordResetConfirm_InvalidToken(t *testing.T) {
	api := newTestAPI(t, "test")

	resp := api.Post("/users/password-reset/confirm", map[string]string{
		"token":    "not-a-real-token",
		"password": "an entirely new password",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestNewCookieOptions_SecureOutsideLocalEnvs(t *testing.T) {
	tests := []struct {
		env        string
		wantSecure bool
	}{
		{"dev", false},
		{"test", false},
		{"stg", true},
		{"prod", true},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			if got := NewCookieOptions(tt.env).Secure; got != tt.wantSecure {
				t.Fatalf("env %q: expected Secure=%v, got %v", tt.env, tt.wantSecure, got)
			}
		})
	}
}

func sessionCookieFrom(t *testing.T, setCookie string) *http.Cookie {
	t.Helper()

	if setCookie == "" {
		t.Fatal("expected a Set-Cookie header")
	}
	header := http.Header{}
	header.Add("Set-Cookie", setCookie)
	for _, c := range (&http.Response{Header: header}).Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	t.Fatalf("no %q cookie in %q", SessionCookieName, setCookie)
	return nil
}
