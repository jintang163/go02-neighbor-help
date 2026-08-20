package handler

import (
	"io/fs"
	"net/http"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/middleware"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
	"go02-neighbor-help/internal/service"
	"go02-neighbor-help/internal/store"
)

type Handler struct {
	services *service.Services
	store    store.Store
	sessions *auth.SessionManager
	assets   fs.FS
}

func New(svc *service.Services, st store.Store, sessions *auth.SessionManager, assets fs.FS) *Handler {
	return &Handler{services: svc, store: st, sessions: sessions, assets: assets}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	authMw := middleware.RequireAuth(h.sessions, h.store)
	admin := middleware.Chain(authMw, middleware.RequireAdmin())
	resident := middleware.Chain(authMw, middleware.RequireRole(model.RoleResident))

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.Handle("POST /api/auth/logout", authMw(http.HandlerFunc(h.Logout)))
	mux.Handle("GET /api/auth/me", authMw(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/me/profile", authMw(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/me/password", authMw(http.HandlerFunc(h.ChangePassword)))

	mux.Handle("GET /api/categories", authMw(http.HandlerFunc(h.Categories)))
	mux.Handle("GET /api/users", admin(http.HandlerFunc(h.ListUsers)))
	mux.Handle("POST /api/users", admin(http.HandlerFunc(h.CreateUser)))
	mux.Handle("GET /api/users/{id}", authMw(http.HandlerFunc(h.GetUser)))
	mux.Handle("GET /api/users/{id}/reviews", authMw(http.HandlerFunc(h.UserReviews)))
	mux.Handle("POST /api/users/{id}/freeze", admin(http.HandlerFunc(h.FreezeUser)))
	mux.Handle("POST /api/users/{id}/unfreeze", admin(http.HandlerFunc(h.UnfreezeUser)))
	mux.Handle("POST /api/users/{id}/credit", admin(http.HandlerFunc(h.AdjustCredit)))

	mux.Handle("GET /api/posts", authMw(http.HandlerFunc(h.ListPosts)))
	mux.Handle("POST /api/posts", resident(http.HandlerFunc(h.CreatePost)))
	mux.Handle("GET /api/posts/{id}", authMw(http.HandlerFunc(h.GetPost)))
	mux.Handle("PUT /api/posts/{id}", authMw(http.HandlerFunc(h.UpdatePost)))
	mux.Handle("POST /api/posts/{id}/publish", authMw(http.HandlerFunc(h.PublishPost)))
	mux.Handle("POST /api/posts/{id}/cancel", authMw(http.HandlerFunc(h.CancelPost)))
	mux.Handle("POST /api/posts/{id}/close", admin(http.HandlerFunc(h.ForceClosePost)))
	mux.Handle("GET /api/me/posts", authMw(http.HandlerFunc(h.MyPosts)))

	mux.Handle("POST /api/posts/{id}/apply", resident(http.HandlerFunc(h.ApplyPost)))
	mux.Handle("GET /api/posts/{id}/applications", authMw(http.HandlerFunc(h.ListPostApplications)))
	mux.Handle("DELETE /api/applications/{id}", authMw(http.HandlerFunc(h.WithdrawApplication)))
	mux.Handle("POST /api/applications/{id}/accept", authMw(http.HandlerFunc(h.AcceptApplication)))
	mux.Handle("POST /api/applications/{id}/reject", authMw(http.HandlerFunc(h.RejectApplication)))
	mux.Handle("GET /api/me/applications", authMw(http.HandlerFunc(h.MyApplications)))

	mux.Handle("GET /api/tasks/{id}", authMw(http.HandlerFunc(h.GetTask)))
	mux.Handle("GET /api/me/tasks", authMw(http.HandlerFunc(h.MyTasks)))
	mux.Handle("POST /api/tasks/{id}/confirm-start", authMw(http.HandlerFunc(h.ConfirmStart)))
	mux.Handle("POST /api/tasks/{id}/complete", authMw(http.HandlerFunc(h.MarkComplete)))
	mux.Handle("POST /api/tasks/{id}/confirm-complete", authMw(http.HandlerFunc(h.ConfirmComplete)))
	mux.Handle("POST /api/tasks/{id}/dispute", authMw(http.HandlerFunc(h.DisputeTask)))
	mux.Handle("POST /api/tasks/{id}/cancel", authMw(http.HandlerFunc(h.CancelTask)))
	mux.Handle("POST /api/tasks/{id}/reviews", authMw(http.HandlerFunc(h.SubmitReview)))
	mux.Handle("GET /api/tasks/{id}/reviews", authMw(http.HandlerFunc(h.ListTaskReviews)))

	mux.Handle("GET /api/posts/{id}/messages", authMw(http.HandlerFunc(h.ListMessages)))
	mux.Handle("POST /api/posts/{id}/messages", authMw(http.HandlerFunc(h.AddMessage)))
	mux.Handle("DELETE /api/messages/{id}", authMw(http.HandlerFunc(h.DeleteMessage)))
	mux.Handle("POST /api/posts/{id}/favorite", resident(http.HandlerFunc(h.Favorite)))
	mux.Handle("DELETE /api/posts/{id}/favorite", resident(http.HandlerFunc(h.Unfavorite)))
	mux.Handle("GET /api/me/favorites", authMw(http.HandlerFunc(h.MyFavorites)))

	mux.Handle("GET /api/me/notifications", authMw(http.HandlerFunc(h.MyNotifications)))
	mux.Handle("POST /api/me/notifications/{id}/read", authMw(http.HandlerFunc(h.ReadNotification)))
	mux.Handle("POST /api/me/notifications/read-all", authMw(http.HandlerFunc(h.ReadAllNotifications)))
	mux.Handle("GET /api/me/credit-logs", authMw(http.HandlerFunc(h.MyCreditLogs)))

	mux.Handle("POST /api/reports", authMw(http.HandlerFunc(h.CreateReport)))
	mux.Handle("GET /api/reports", admin(http.HandlerFunc(h.ListReports)))
	mux.Handle("POST /api/reports/{id}/handle", admin(http.HandlerFunc(h.HandleReport)))
	mux.Handle("GET /api/stats", admin(http.HandlerFunc(h.GlobalStats)))

	h.registerPageRoutes(mux)
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, model.HealthResponse{Status: "ok"})
}
