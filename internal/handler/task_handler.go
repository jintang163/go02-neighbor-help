package handler

import (
	"net/http"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
)

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Task.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyTasks(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Task.Mine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) ConfirmStart(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Task.ConfirmStart(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MarkComplete(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Task.MarkComplete(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ConfirmComplete(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Task.ConfirmComplete(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) DisputeTask(w http.ResponseWriter, r *http.Request) {
	var in model.DisputeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Task.Dispute(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	var in model.CancelInput
	if r.ContentLength != 0 && !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Task.Cancel(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	var in model.ReviewInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Review.Submit(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListTaskReviews(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Review.ListByTask(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}
