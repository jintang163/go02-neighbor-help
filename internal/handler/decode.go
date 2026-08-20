package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
	"go02-neighbor-help/internal/service"
)

const maxBodySize = 1 << 20

const bearerPrefix = "Bearer "

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(h[len(bearerPrefix):])
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		switch {
		case errors.Is(err, io.EOF):
			respond.Error(w, http.StatusBadRequest, "bad_request", "请求体为空")
		case strings.Contains(err.Error(), "unknown field"):
			respond.Error(w, http.StatusBadRequest, "bad_request", "请求包含未知字段: "+err.Error())
		default:
			respond.Error(w, http.StatusBadRequest, "bad_request", "请求体格式错误: "+err.Error())
		}
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		respond.Error(w, http.StatusBadRequest, "bad_request", "请求体只能包含一个 JSON 对象")
		return false
	}
	return true
}

func queryStr(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func pathID(r *http.Request) string { return r.PathValue("id") }

func userFrom(r *http.Request) model.User {
	u, _ := service.UserFromContext(r.Context())
	return u
}

func writeErr(w http.ResponseWriter, err error) {
	respond.DomainError(w, err)
}

func parseBoolQuery(r *http.Request, key string, def bool) bool {
	s := strings.ToLower(queryStr(r, key))
	switch s {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return def
	}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
