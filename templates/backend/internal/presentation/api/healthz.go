package api

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

type healthzOutput struct {
	Body struct {
		Status string `json:"status" example:"ok"`
	}
}

// RegisterHealthz registers the cross-cutting GET /healthz liveness
// check. No domain owns it.
func RegisterHealthz(api huma.API, logger *slog.Logger) {
	g := httpx.NewGroup(api, "", "System", logger)

	httpx.Get(g, "/healthz", "healthz", getHealthz,
		httpx.Summary("Health check"))
}

func getHealthz(_ *httpx.Ctx, _ *struct{}) (*healthzOutput, error) {
	out := &healthzOutput{}
	out.Body.Status = "ok"
	return out, nil
}
