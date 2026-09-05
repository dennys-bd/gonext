package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"golden-app/backend/internal/presentation/httpx"
)

const readyzTimeout = 2 * time.Second

// pinger is satisfied by *sql.DB and *bun.DB (which embeds *sql.DB),
// so /readyz can depend on the narrow capability it needs instead of
// a concrete database type.
type pinger interface {
	PingContext(ctx context.Context) error
}

type readyzOutput struct {
	Status int
	Body   struct {
		Status string `json:"status" example:"ok"`
	}
}

// RegisterReadyz registers the cross-cutting GET /readyz readiness
// check, which pings db to confirm the server can actually serve
// DB-backed requests. Unlike /healthz, a failure here is non-fatal:
// it is reported through the output's own Status field rather than
// the error return, so it renders 503 and callers back off and retry
// — the group wrapper only post-processes a non-nil error, so this
// path is untouched by it.
func RegisterReadyz(api huma.API, db pinger, logger *slog.Logger) {
	g := httpx.NewGroup(api, "", "System", logger)

	httpx.Get(g, "/readyz", "readyz", func(rctx *httpx.Ctx, _ *struct{}) (*readyzOutput, error) {
		ctx, cancel := context.WithTimeout(rctx, readyzTimeout)
		defer cancel()

		out := &readyzOutput{}
		if err := db.PingContext(ctx); err != nil {
			out.Status = http.StatusServiceUnavailable
			out.Body.Status = "unavailable"
			return out, nil
		}
		out.Status = http.StatusOK
		out.Body.Status = "ok"
		return out, nil
	}, httpx.Summary("Readiness check"))
}
