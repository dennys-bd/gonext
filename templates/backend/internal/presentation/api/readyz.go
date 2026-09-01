package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
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
// it returns 503 so callers back off and retry.
func RegisterReadyz(api huma.API, db pinger) {
	httpx.Register(api, huma.Operation{
		OperationID: "readyz",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Readiness check",
		Tags:        []string{"System"},
	}, func(rctx *httpx.Ctx, _ *struct{}) (*readyzOutput, error) {
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
	})
}
