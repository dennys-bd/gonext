package api

import (
	"net/http"

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
func RegisterHealthz(api huma.API) {
	httpx.Register(api, huma.Operation{
		OperationID: "healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Health check",
		Tags:        []string{"System"},
	}, func(_ *httpx.Ctx, _ *struct{}) (*healthzOutput, error) {
		out := &healthzOutput{}
		out.Body.Status = "ok"
		return out, nil
	})
}
