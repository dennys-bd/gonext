// Package presentation exposes the users domain's HTTP operations on
// a shared huma.API, translating domain errors into HTTP status codes
// and owning the session cookie mechanics.
package presentation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/internal/presentation/httpx"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/application"
)

// checkYourEmail is the deliberately vague response both registration
// branches and every password reset request return, so a caller
// cannot learn which emails are registered.
const checkYourEmail = "If that email needs action, we've sent a message to it."

type credentialsInput struct {
	Body struct {
		Email    string `json:"email" doc:"Account email address" example:"ada@example.com"`
		Password string `json:"password" doc:"Account password" example:"correct horse battery staple"`
	}
}

type acceptedOutput struct {
	Body struct {
		Message string `json:"message"`
		// Token is the raw one-shot token, returned only outside
		// production so local development and the smoke tests can
		// complete the flow without a mailer.
		Token string `json:"token,omitempty"`
	}
}

type messageOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type userOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      userBody
}

type meOutput struct {
	Body userBody
}

type userBody struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	Permissions   []string `json:"permissions"`
	EmailVerified bool     `json:"emailVerified"`
}

type logoutInput struct {
	Session string `cookie:"session" doc:"Session cookie issued by POST /users/login"`
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type confirmEmailInput struct {
	Body struct {
		Token string `json:"token" doc:"Email confirmation token"`
	}
}

type passwordResetInput struct {
	Body struct {
		Email string `json:"email" doc:"Account email address" example:"ada@example.com"`
	}
}

type passwordResetConfirmInput struct {
	Body struct {
		Token    string `json:"token" doc:"Password reset token"`
		Password string `json:"password" doc:"New account password"`
	}
}

// RegisterUsers registers the users domain's endpoints on api, backed
// by svc. cookies carries the session cookie policy, and logger
// records the unexpected errors that are deliberately not returned to
// the client.
func RegisterUsers(api huma.API, svc *application.UserService, cookies CookieOptions, logger *slog.Logger) {
	fail := errorTranslator(logger)
	httpx.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/users/register",
		Summary:       "Register an account",
		Description:   "Always responds the same way whether or not the email was already registered.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx *httpx.Ctx, input *credentialsInput) (*acceptedOutput, error) {
		res, err := svc.Register(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, fail(ctx, err)
		}
		out := &acceptedOutput{}
		out.Body.Message, out.Body.Token = checkYourEmail, res.DevToken
		return out, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID: "login-user",
		Method:      http.MethodPost,
		Path:        "/users/login",
		Summary:     "Log in",
		Description: "Issues a session and sets the session cookie.",
		Tags:        []string{"Users"},
	}, func(ctx *httpx.Ctx, input *credentialsInput) (*userOutput, error) {
		token, expiresAt, err := svc.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, fail(ctx, err)
		}

		identity, user, err := svc.Me(ctx, token)
		if err != nil {
			return nil, fail(ctx, err)
		}

		return &userOutput{
			SetCookie: cookies.sessionCookie(token, expiresAt),
			Body:      toUserBody(identity, user),
		}, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID:   "logout-user",
		Method:        http.MethodPost,
		Path:          "/users/logout",
		Summary:       "Log out",
		Description:   "Revokes the current session and clears the session cookie.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
		Security:      auth.Required(),
	}, func(ctx *httpx.Ctx, input *logoutInput) (*logoutOutput, error) {
		if err := svc.Logout(ctx, input.Session); err != nil {
			return nil, fail(ctx, err)
		}
		return &logoutOutput{SetCookie: cookies.clearedSessionCookie()}, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID: "get-current-user",
		Method:      http.MethodGet,
		Path:        "/users/me",
		Summary:     "Get the current user",
		Tags:        []string{"Users"},
		Security:    auth.Required(),
	}, func(ctx *httpx.Ctx, _ *struct{}) (*meOutput, error) {
		identity := ctx.Identity()
		user, err := svc.Profile(ctx, identity.UserID)
		if err != nil {
			return nil, fail(ctx, err)
		}
		return &meOutput{Body: toUserBody(identity, user)}, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID: "confirm-user-email",
		Method:      http.MethodPost,
		Path:        "/users/confirm-email",
		Summary:     "Confirm an email address",
		Tags:        []string{"Users"},
	}, func(ctx *httpx.Ctx, input *confirmEmailInput) (*messageOutput, error) {
		if err := svc.ConfirmEmail(ctx, input.Body.Token); err != nil {
			return nil, fail(ctx, err)
		}
		out := &messageOutput{}
		out.Body.Message = "Email confirmed."
		return out, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID:   "request-password-reset",
		Method:        http.MethodPost,
		Path:          "/users/password-reset",
		Summary:       "Request a password reset",
		Description:   "Always responds the same way whether or not the email is registered.",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx *httpx.Ctx, input *passwordResetInput) (*acceptedOutput, error) {
		devToken, err := svc.RequestPasswordReset(ctx, input.Body.Email)
		if err != nil {
			return nil, fail(ctx, err)
		}
		out := &acceptedOutput{}
		out.Body.Message, out.Body.Token = checkYourEmail, devToken
		return out, nil
	})

	httpx.Register(api, huma.Operation{
		OperationID: "confirm-password-reset",
		Method:      http.MethodPost,
		Path:        "/users/password-reset/confirm",
		Summary:     "Set a new password with a reset token",
		Tags:        []string{"Users"},
	}, func(ctx *httpx.Ctx, input *passwordResetConfirmInput) (*messageOutput, error) {
		if err := svc.ConfirmPasswordReset(ctx, input.Body.Token, input.Body.Password); err != nil {
			return nil, fail(ctx, err)
		}
		out := &messageOutput{}
		out.Body.Message = "Password updated."
		return out, nil
	})
}

// errorTranslator builds the handler-facing error mapper. Unhandled
// errors are logged and replaced with a flat message: huma renders a
// non-StatusError by putting err.Error() into the response body, so
// returning the raw error would ship wrapped internals — including
// database driver text — to the client.
func errorTranslator(logger *slog.Logger) func(context.Context, error) error {
	return func(ctx context.Context, err error) error {
		if translated, ok := toHTTPError(err); ok {
			return translated
		}
		logger.ErrorContext(ctx, "users: unhandled error", "error", err)
		return huma.Error500InternalServerError("internal server error")
	}
}

// toHTTPError maps the users domain's sentinel errors onto status
// codes, reporting whether err was one it recognises.
func toHTTPError(err error) (error, bool) {
	switch {
	case errors.Is(err, domain.ErrEmailRequired),
		errors.Is(err, domain.ErrPasswordTooShort),
		errors.Is(err, domain.ErrConfirmationTokenInvalid),
		errors.Is(err, domain.ErrResetTokenInvalid):
		return huma.Error400BadRequest(err.Error()), true
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrSessionInvalid):
		return huma.Error401Unauthorized(err.Error()), true
	case errors.Is(err, domain.ErrEmailNotConfirmed):
		return huma.Error403Forbidden(err.Error()), true
	case errors.Is(err, domain.ErrEmailTaken):
		return huma.Error409Conflict(err.Error()), true
	default:
		return nil, false
	}
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
