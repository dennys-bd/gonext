// Package presentation exposes the users domain's HTTP operations on
// a shared huma.API, translating domain errors into HTTP status codes
// and owning the session cookie mechanics.
package presentation

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/internal/presentation/httpx"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/application"
)

// checkYourEmail is the deliberately vague response returned by a
// successful registration and by every password reset request, so the
// reset endpoint cannot be used to learn which emails are registered.
// Registration itself answers 409 for an address already in use — that
// enumeration trade-off was accepted deliberately; see the users auth
// and RBAC design.
const checkYourEmail = "If that email needs action, we've sent a message to it."

type handlers struct {
	svc     *application.UserService
	cookies CookieOptions
}

// RegisterUsers registers the users domain's endpoints on api, backed
// by svc. cookies carries the session cookie policy, and logger
// records the unexpected errors that are deliberately not returned to
// the client.
func RegisterUsers(api huma.API, svc *application.UserService, cookies CookieOptions, logger *slog.Logger) {
	h := &handlers{svc: svc, cookies: cookies}
	g := httpx.NewGroup(api, "/users", "Users", logger).Errors(
		httpx.Map(domain.ErrEmailRequired, http.StatusBadRequest),
		httpx.Map(domain.ErrPasswordTooShort, http.StatusBadRequest),
		httpx.Map(domain.ErrConfirmationTokenInvalid, http.StatusBadRequest),
		httpx.Map(domain.ErrResetTokenInvalid, http.StatusBadRequest),
		httpx.Map(domain.ErrInvalidCredentials, http.StatusUnauthorized),
		httpx.Map(domain.ErrSessionInvalid, http.StatusUnauthorized),
		httpx.Map(domain.ErrEmailNotConfirmed, http.StatusForbidden),
		httpx.Map(domain.ErrEmailTaken, http.StatusConflict),
	)

	httpx.Post(g, "/register", "register-user", h.register,
		httpx.Summary("Register an account"),
		httpx.Description("Responds 202 with a deliberately generic message. An address that is already registered is rejected with 409, so this endpoint does not conceal whether an email exists."),
		httpx.Status(http.StatusAccepted))

	httpx.Post(g, "/login", "login-user", h.login,
		httpx.Summary("Log in"),
		httpx.Description("Issues a session and sets the session cookie."))

	httpx.Post(g, "/logout", "logout-user", h.logout,
		httpx.Summary("Log out"),
		httpx.Description("Revokes the current session and clears the session cookie."),
		httpx.Status(http.StatusNoContent),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "/me", "get-current-user", h.me,
		httpx.Summary("Get the current user"),
		httpx.Secured(auth.Required()))

	httpx.Post(g, "/confirm-email", "confirm-user-email", h.confirmEmail,
		httpx.Summary("Confirm an email address"))

	httpx.Post(g, "/password-reset", "request-password-reset", h.requestPasswordReset,
		httpx.Summary("Request a password reset"),
		httpx.Description("Always responds the same way whether or not the email is registered."),
		httpx.Status(http.StatusAccepted))

	httpx.Post(g, "/password-reset/confirm", "confirm-password-reset", h.confirmPasswordReset,
		httpx.Summary("Set a new password with a reset token"))
}

func (h *handlers) register(ctx *httpx.Ctx, in *credentialsInput) (*acceptedOutput, error) {
	res, err := h.svc.Register(ctx, in.Body.Email, in.Body.Password)
	if err != nil {
		return nil, err
	}
	out := &acceptedOutput{}
	out.Body.Message, out.Body.Token = checkYourEmail, res.DevToken
	return out, nil
}

func (h *handlers) login(ctx *httpx.Ctx, in *credentialsInput) (*userOutput, error) {
	token, expiresAt, err := h.svc.Login(ctx, in.Body.Email, in.Body.Password)
	if err != nil {
		return nil, err
	}

	identity, user, err := h.svc.Me(ctx, token)
	if err != nil {
		return nil, err
	}

	return &userOutput{
		SetCookie: h.cookies.sessionCookie(token, expiresAt),
		Body:      toUserBody(identity, user),
	}, nil
}

func (h *handlers) logout(ctx *httpx.Ctx, in *logoutInput) (*logoutOutput, error) {
	if err := h.svc.Logout(ctx, in.Session); err != nil {
		return nil, err
	}
	return &logoutOutput{SetCookie: h.cookies.clearedSessionCookie()}, nil
}

func (h *handlers) me(ctx *httpx.Ctx, _ *struct{}) (*meOutput, error) {
	identity := ctx.Identity()
	user, err := h.svc.Profile(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &meOutput{Body: toUserBody(identity, user)}, nil
}

func (h *handlers) confirmEmail(ctx *httpx.Ctx, in *confirmEmailInput) (*messageOutput, error) {
	if err := h.svc.ConfirmEmail(ctx, in.Body.Token); err != nil {
		return nil, err
	}
	out := &messageOutput{}
	out.Body.Message = "Email confirmed."
	return out, nil
}

func (h *handlers) requestPasswordReset(ctx *httpx.Ctx, in *passwordResetInput) (*acceptedOutput, error) {
	devToken, err := h.svc.RequestPasswordReset(ctx, in.Body.Email)
	if err != nil {
		return nil, err
	}
	out := &acceptedOutput{}
	out.Body.Message, out.Body.Token = checkYourEmail, devToken
	return out, nil
}

func (h *handlers) confirmPasswordReset(ctx *httpx.Ctx, in *passwordResetConfirmInput) (*messageOutput, error) {
	if err := h.svc.ConfirmPasswordReset(ctx, in.Body.Token, in.Body.Password); err != nil {
		return nil, err
	}
	out := &messageOutput{}
	out.Body.Message = "Password updated."
	return out, nil
}

func toUserBody(identity auth.Identity, user domain.User) userBody {
	permissions := identity.Permissions
	if permissions == nil {
		permissions = []string{}
	}
	return userBody{
		ID:            user.ID,
		Email:         user.Email,
		Role:          user.Role,
		Permissions:   permissions,
		EmailVerified: user.EmailVerified(),
	}
}
