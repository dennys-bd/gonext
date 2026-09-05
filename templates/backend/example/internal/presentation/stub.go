// Package presentation exposes the example domain's HTTP operations on
// a shared huma.API, translating domain errors into HTTP status codes.
package presentation

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"

	"[PROJECT-NAME]/backend/example/domain"
	"[PROJECT-NAME]/backend/example/internal/application"
	"[PROJECT-NAME]/backend/internal/presentation/httpx"
)

type createStubInput struct {
	Body struct {
		Name string `json:"name" doc:"Name for the stub" example:"my-stub"`
	}
}

type stubOutput struct {
	Body struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
	}
}

type getStubInput struct {
	ID string `path:"id" doc:"Stub id"`
}

type handlers struct {
	svc *application.StubService
}

// RegisterStub registers POST /stubs and GET /stubs/{id} on api, backed
// by svc. Unmapped errors are logged through logger and returned to
// the client as a flat 500.
func RegisterStub(api huma.API, svc *application.StubService, logger *slog.Logger) {
	h := &handlers{svc: svc}
	g := httpx.NewGroup(api, "/stubs", "Example", logger).Errors(
		httpx.Map(domain.ErrStubNameRequired, http.StatusBadRequest),
		httpx.Map(domain.ErrStubNotFound, http.StatusNotFound),
	)

	httpx.Post(g, "", "create-stub", h.createStub,
		httpx.Summary("Create a stub"),
		httpx.Status(http.StatusCreated),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "/{id}", "get-stub", h.getStub,
		httpx.Summary("Get a stub by id"))
}

func (h *handlers) createStub(ctx *httpx.Ctx, in *createStubInput) (*stubOutput, error) {
	stub, err := h.svc.CreateStub(ctx, in.Body.Name)
	if err != nil {
		return nil, err
	}
	return toStubOutput(stub), nil
}

func (h *handlers) getStub(ctx *httpx.Ctx, in *getStubInput) (*stubOutput, error) {
	stub, err := h.svc.GetStub(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return toStubOutput(stub), nil
}

func toStubOutput(s domain.Stub) *stubOutput {
	out := &stubOutput{}
	out.Body.ID, out.Body.Name, out.Body.CreatedAt = s.ID, s.Name, s.CreatedAt
	return out
}
