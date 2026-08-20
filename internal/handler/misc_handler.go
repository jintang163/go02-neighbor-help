package handler

import (
	"net/http"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
)

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Social.ListMessages(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) AddMessage(w http.ResponseWriter, r *http.Request) {
	var in model.MessageInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Social.AddMessage(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	if err := h.services.Social.DeleteMessage(r.Context(), userFrom(r), pathID(r)); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Social.Favorite(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) Unfavorite(w http.ResponseWriter, r *http.Request) {
	if err := h.services.Social.Unfavorite(r.Context(), userFrom(r), pathID(r)); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) MyFavorites(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.Social.MyFavorites(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) MyNotifications(w http.ResponseWriter, r *http.Request) {
	unread := parseBoolQuery(r, "unread", false)
	items, err := h.services.Notify.List(r.Context(), userFrom(r).ID, unread)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Notify.MarkRead(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	n, err := h.services.Notify.MarkAllRead(r.Context(), userFrom(r).ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]int{"updated": n})
}

func (h *Handler) MyCreditLogs(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	items, err := h.services.User.CreditLogs(r.Context(), u, u.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var in model.ReportInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Report.Create(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	status := model.ReportStatus(queryStr(r, "status"))
	items, err := h.services.Report.List(r.Context(), userFrom(r), status)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	var in model.HandleReportInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Report.Handle(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) GlobalStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Stats.Global(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}
