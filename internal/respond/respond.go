package respond

import (
	"encoding/json"
	"net/http"

	"go02-neighbor-help/internal/model"
)

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func OK(w http.ResponseWriter, body any) {
	JSON(w, http.StatusOK, body)
}

func Created(w http.ResponseWriter, body any) {
	JSON(w, http.StatusCreated, body)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, model.ErrorResponse{Code: code, Message: message})
}

func DomainError(w http.ResponseWriter, err error) {
	switch {
	case model.IsNotFound(err):
		Error(w, http.StatusNotFound, "not_found", err.Error())
	case model.IsUnauthorized(err), model.IsInvalidCredentials(err):
		Error(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case model.IsForbidden(err):
		Error(w, http.StatusForbidden, "forbidden", err.Error())
	case model.IsAlreadyExists(err):
		Error(w, http.StatusConflict, "already_exists", err.Error())
	case model.IsConflict(err):
		Error(w, http.StatusConflict, "conflict", err.Error())
	case model.IsValidation(err):
		Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}
