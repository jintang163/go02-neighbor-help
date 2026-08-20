package handler

import (
	"net/http"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in model.UserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Auth.Register(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in model.LoginInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Auth.Login(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.services.Auth.Logout(extractBearer(r))
	respond.NoContent(w)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Auth.Me(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in model.ProfileInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.UpdateProfile(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in model.PasswordInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.services.Auth.ChangePassword(r.Context(), userFrom(r), in); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	f := model.UserFilter{
		Role:   model.UserRole(queryStr(r, "role")),
		Status: model.UserStatus(queryStr(r, "status")),
		Query:  queryStr(r, "q"),
	}
	items, err := h.services.User.List(r.Context(), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in model.UserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.CreateResident(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.PublicProfile(r.Context(), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UserReviews(w http.ResponseWriter, r *http.Request) {
	items, err := h.services.User.ReviewsReceived(r.Context(), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, model.NewList(items))
}

func (h *Handler) FreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Freeze(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UnfreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Unfreeze(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) AdjustCredit(w http.ResponseWriter, r *http.Request) {
	var in model.CreditAdjustInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.AdjustCredit(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, model.DefaultDict())
}
