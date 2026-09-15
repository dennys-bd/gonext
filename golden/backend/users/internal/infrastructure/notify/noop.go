// Package notify provides Core's default domain.Notifier: one that
// logs instead of sending mail. A real implementation arrives with
// the transactional email feature pack.
package notify

import (
	"context"
	"log/slog"

	"golden-app/backend/users/domain"
)

var _ domain.Notifier = (*Noop)(nil)

// Noop is a domain.Notifier that delivers nothing and logs each message at debug
// level, including the raw token — the only way to complete flows locally without
// a mailer. Keep debug logging off in production.
type Noop struct {
	logger *slog.Logger
}

// NewNoop constructs a Noop logging to logger.
func NewNoop(logger *slog.Logger) *Noop {
	return &Noop{logger: logger}
}

// SendConfirmation logs the email confirmation that would have been sent.
func (n *Noop) SendConfirmation(ctx context.Context, email, token string) error {
	n.logger.DebugContext(ctx, "users: would send email confirmation", "email", email, "token", token)
	return nil
}

// SendAccountExistsNotice logs the account-exists notice that would
// have been sent.
func (n *Noop) SendAccountExistsNotice(ctx context.Context, email, resetToken string) error {
	n.logger.DebugContext(ctx, "users: would send account exists notice", "email", email, "token", resetToken)
	return nil
}
