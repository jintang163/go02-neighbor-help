package handler

import (
	"net/http"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
)

func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	f := model.PostFilter{
		Type:     model.PostType(queryStr(r, "type")),
		Status:   model.PostStatus(queryStr(r, "status")),
		Category: model.Category(queryStr(r, "category")),
		Urgency:  model.Urgency(queryStr(r, "urgency")),
		Building: queryStr(r, "building"),
		AuthorID: queryStr(r, "author_id"),
		Query:    queryStr(r, "q"),
		Plaza:    parseBoolQuery(r, "plaza", true),
	}
	items, err := h.services.Post.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var in model.PostInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Post.Create(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Post.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	var in model.PostInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Post.Update(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) PublishPost(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Post.Publish(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CancelPost(w http.ResponseWriter, r *http.Request) {
	var in model.CancelInput
	if r.ContentLength != 0 && !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Post.Cancel(r.Context(), userFrom(r), pathID(r), in.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ForceClosePost(w http.ResponseWriter, r *http.Request) {
	var in model.CancelInput
	if r.ContentLength != 0 && !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Post.ForceClose(r.Context(), userFrom(r), pathID(r), in.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyPosts(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Post.Mine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) ApplyPost(w http.ResponseWriter, r *http.Request) {
	var in model.ApplyInput
	if r.ContentLength != 0 && !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Match.Apply(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListPostApplications(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Match.ListByPost(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) WithdrawApplication(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Match.Withdraw(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) AcceptApplication(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Match.Accept(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Match.Reject(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyApplications(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Match.Mine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}
