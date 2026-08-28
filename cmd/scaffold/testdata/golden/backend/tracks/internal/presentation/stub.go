// Package presentation exposes the tracks domain's HTTP operations on
// a shared huma.API, translating domain errors into HTTP status codes.
package presentation

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"golden-app/backend/tracks/domain"
	"golden-app/backend/tracks/internal/application"
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

// RegisterStub registers POST /stubs and GET /stubs/{id} on api, backed
// by svc.
func RegisterStub(api huma.API, svc *application.StubService) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-stub",
		Method:        http.MethodPost,
		Path:          "/stubs",
		Summary:       "Create a stub",
		Tags:          []string{"Tracks"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *createStubInput) (*stubOutput, error) {
		stub, err := svc.CreateStub(ctx, input.Body.Name)
		if err != nil {
			if errors.Is(err, domain.ErrStubNameRequired) {
				return nil, huma.Error400BadRequest(err.Error())
			}
			return nil, err
		}
		return toStubOutput(stub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-stub",
		Method:      http.MethodGet,
		Path:        "/stubs/{id}",
		Summary:     "Get a stub by id",
		Tags:        []string{"Tracks"},
	}, func(ctx context.Context, input *getStubInput) (*stubOutput, error) {
		stub, err := svc.GetStub(ctx, input.ID)
		if err != nil {
			if errors.Is(err, domain.ErrStubNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, err
		}
		return toStubOutput(stub), nil
	})
}

func toStubOutput(s domain.Stub) *stubOutput {
	out := &stubOutput{}
	out.Body.ID, out.Body.Name, out.Body.CreatedAt = s.ID, s.Name, s.CreatedAt
	return out
}
