package memory

import (
	"context"
	"sync"

	"[PROJECT-NAME]/backend/users/domain"
)

var _ domain.Notifier = (*Notifier)(nil)

// Notification is one message a Notifier was asked to deliver.
type Notification struct {
	Kind  string
	Email string
	Token string
}

// Notifier is a domain.Notifier that records what it was asked to
// send instead of sending it, so tests can assert which branch of a
// use case ran.
type Notifier struct {
	mu   sync.Mutex
	sent []Notification
}

// NewNotifier constructs an empty recording Notifier.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// SendConfirmation records a confirmation notification.
func (n *Notifier) SendConfirmation(_ context.Context, email, token string) error {
	n.record(Notification{Kind: "confirmation", Email: email, Token: token})
	return nil
}

// SendAccountExistsNotice records an account-exists notification.
func (n *Notifier) SendAccountExistsNotice(_ context.Context, email, resetToken string) error {
	n.record(Notification{Kind: "account-exists", Email: email, Token: resetToken})
	return nil
}

// Sent returns a copy of everything recorded so far.
func (n *Notifier) Sent() []Notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Notification(nil), n.sent...)
}

func (n *Notifier) record(msg Notification) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, msg)
}
