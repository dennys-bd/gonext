package presentation

// The request and response shapes for the users track's HTTP
// operations. They live beside users.go rather than in it so the route
// surface and its handlers read without scrolling past seventy lines of
// struct tags.

import "net/http"

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
