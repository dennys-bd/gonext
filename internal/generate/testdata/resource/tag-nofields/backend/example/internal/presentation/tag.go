package presentation

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/application"
	"golden-app/backend/internal/presentation/httpx"
)

// tagBody is the request body shared by create and update.
type tagBody struct{}

type tagItem struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type createTagInput struct {
	Body tagBody
}

type tagIDInput struct {
	ID string `path:"id" doc:"Tag id"`
}

type listTagsInput struct {
	Limit  int `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int `query:"offset" minimum:"0"`
}

type updateTagInput struct {
	ID   string `path:"id" doc:"Tag id"`
	Body tagBody
}

type tagOutput struct {
	Body tagItem
}

type listTagsOutput struct {
	Body struct {
		Items []tagItem `json:"items"`
	}
}

type tagHandlers struct {
	svc *application.TagService
}

// RegisterTag registers the tag endpoints under /tags on api,
// backed by svc. Unmapped errors are logged through logger and returned
// to the client as a flat 500.
func RegisterTag(api huma.API, svc *application.TagService, logger *slog.Logger) {
	h := &tagHandlers{svc: svc}
	g := httpx.NewGroup(api, "/tags", "Example", logger).Errors(
		httpx.Map(domain.ErrTagNotFound, http.StatusNotFound),
	)

	httpx.Post(g, "", "create-tag", h.createTag,
		httpx.Summary("Create a tag"),
		httpx.Status(http.StatusCreated),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "/{id}", "get-tag", h.getTag,
		httpx.Summary("Get a tag by id"),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "", "list-tags", h.listTags,
		httpx.Summary("List tags"),
		httpx.Secured(auth.Required()))

	httpx.Put(g, "/{id}", "update-tag", h.updateTag,
		httpx.Summary("Replace a tag"),
		httpx.Secured(auth.Required()))

	httpx.Delete(g, "/{id}", "delete-tag", h.deleteTag,
		httpx.Summary("Delete a tag"),
		httpx.Status(http.StatusNoContent),
		httpx.Secured(auth.Required()))
}

func (h *tagHandlers) createTag(ctx *httpx.Ctx, _ *createTagInput) (*tagOutput, error) {
	tag, err := h.svc.CreateTag(ctx)
	if err != nil {
		return nil, err
	}
	return &tagOutput{Body: toTagItem(tag)}, nil
}

func (h *tagHandlers) getTag(ctx *httpx.Ctx, in *tagIDInput) (*tagOutput, error) {
	tag, err := h.svc.GetTag(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &tagOutput{Body: toTagItem(tag)}, nil
}

func (h *tagHandlers) listTags(ctx *httpx.Ctx, in *listTagsInput) (*listTagsOutput, error) {
	tags, err := h.svc.ListTags(ctx, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	out := &listTagsOutput{}
	out.Body.Items = make([]tagItem, 0, len(tags))
	for _, tag := range tags {
		out.Body.Items = append(out.Body.Items, toTagItem(tag))
	}
	return out, nil
}

func (h *tagHandlers) updateTag(ctx *httpx.Ctx, in *updateTagInput) (*tagOutput, error) {
	tag, err := h.svc.UpdateTag(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &tagOutput{Body: toTagItem(tag)}, nil
}

func (h *tagHandlers) deleteTag(ctx *httpx.Ctx, in *tagIDInput) (*struct{}, error) {
	if err := h.svc.DeleteTag(ctx, in.ID); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func toTagItem(tag domain.Tag) tagItem {
	return tagItem{
		ID:        tag.ID,
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	}
}
