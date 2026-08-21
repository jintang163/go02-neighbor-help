package service

import (
	"context"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type ReportService struct {
	store    store.Store
	sessions *auth.SessionManager
	credit   *CreditHelper
	notify   *NotifyService
	clock    Clock
}

func NewReportService(s store.Store, sessions *auth.SessionManager, credit *CreditHelper, notify *NotifyService, clock Clock) *ReportService {
	return &ReportService{store: s, sessions: sessions, credit: credit, notify: notify, clock: clock}
}

func (r *ReportService) Create(ctx context.Context, actor model.User, in model.ReportInput) (model.Report, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Report{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Report{}, err
	}
	if err := r.ensureTarget(ctx, in); err != nil {
		return model.Report{}, err
	}
	if in.TargetType == model.ReportUser && in.TargetID == actor.ID {
		return model.Report{}, model.ErrValidation
	}
	return r.store.CreateReport(ctx, model.Report{
		ReporterID: actor.ID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Reason:     in.Reason,
		Detail:     in.Detail,
	})
}

func (r *ReportService) List(ctx context.Context, actor model.User, status model.ReportStatus) ([]model.ReportView, error) {
	if !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	items, err := r.store.ListReports(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]model.ReportView, 0, len(items))
	for _, item := range items {
		view := model.ReportView{Report: item}
		if u, err := publicOf(ctx, r.store, item.ReporterID); err == nil {
			view.Reporter = u
		}
		out = append(out, view)
	}
	return out, nil
}

func (r *ReportService) Handle(ctx context.Context, actor model.User, id string, in model.HandleReportInput) (model.Report, error) {
	if !actor.IsAdmin() {
		return model.Report{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Report{}, err
	}
	rep, err := r.store.GetReport(ctx, id)
	if err != nil {
		return model.Report{}, err
	}
	if rep.Status != model.ReportPending {
		return model.Report{}, model.ErrConflict
	}
	now := r.clock.Now()
	rep.HandlerID = actor.ID
	rep.HandleNote = in.Note
	rep.HandledAt = &now
	if in.Action == "accept" {
		rep.Status = model.ReportAccepted
		if err := r.applyAccept(ctx, actor, rep, in.Freeze); err != nil {
			return model.Report{}, err
		}
	} else {
		rep.Status = model.ReportRejected
	}
	updated, err := r.store.UpdateReport(ctx, rep)
	if err != nil {
		return model.Report{}, err
	}
	r.notify.Push(ctx, rep.ReporterID, model.NotifyReportHandled, "举报已处理", "处理结果："+string(updated.Status), updated.ID, "report")
	return updated, nil
}

func (r *ReportService) ensureTarget(ctx context.Context, in model.ReportInput) error {
	switch in.TargetType {
	case model.ReportUser:
		_, err := r.store.GetUserByID(ctx, in.TargetID)
		return err
	case model.ReportPost:
		_, err := r.store.GetPost(ctx, in.TargetID)
		return err
	case model.ReportReview:
		_, err := r.store.GetReview(ctx, in.TargetID)
		return err
	case model.ReportMessage:
		_, err := r.store.GetMessage(ctx, in.TargetID)
		return err
	default:
		return model.ErrValidation
	}
}

func (r *ReportService) applyAccept(ctx context.Context, _ model.User, rep model.Report, freeze bool) error {
	switch rep.TargetType {
	case model.ReportUser:
		u, err := r.store.GetUserByID(ctx, rep.TargetID)
		if err != nil {
			return err
		}
		// 信用账本写入是举报成立流程中最易失败的一步，先做：失败时尚未冻结用户、
		// 未清除登录会话，举报维持 pending，可安全重试。返回的 u 携带最新信用分，
		// 后续冻结据此回写，避免用过期副本覆盖刚落账的信用分。
		updated, err := r.credit.Apply(ctx, u.ID, -10, model.CreditReportAccepted, rep.ID, "举报成立")
		if err != nil {
			return err
		}
		u = updated
		if freeze && !u.IsAdmin() {
			u.Status = model.UserFrozen
			if _, err := r.store.UpdateUser(ctx, u); err != nil {
				// 冻结失败：冲销已扣减的信用分，使举报、用户与会话保持一致且可重试。
				_, _ = r.credit.Apply(ctx, u.ID, 10, model.CreditReportAccepted, rep.ID, "举报成立（回滚）")
				return err
			}
			r.sessions.InvalidateByUser(u.ID)
		}
		return nil
	case model.ReportPost:
		post, err := r.store.GetPost(ctx, rep.TargetID)
		if err != nil {
			return err
		}
		// 信用账本写入是举报成立流程中最易失败的一步，先做：失败时帖子尚未关闭，
		// 举报维持 pending，可安全重试。成功后再关闭帖子；若关闭失败则冲销已扣减的
		// 信用分，使举报与帖子保持原状态且可重试，避免帖子已提前关闭而信用未扣
		// 造成的状态不一致。
		if _, err := r.credit.Apply(ctx, post.AuthorID, -10, model.CreditReportAccepted, rep.ID, "帖子举报成立"); err != nil {
			return err
		}
		if !post.Status.IsTerminal() {
			post.Status = model.PostClosed
			post.ClosedReason = "report accepted"
			if _, err := r.store.UpdatePost(ctx, post); err != nil {
				// 帖子关闭失败：冲销已扣减的信用分，使举报维持 pending 且帖子保持原状态，可安全重试。
				_, _ = r.credit.Apply(ctx, post.AuthorID, 10, model.CreditReportAccepted, rep.ID, "帖子举报成立（回滚）")
				return err
			}
		}
		return nil
	case model.ReportMessage:
		return r.store.DeleteMessage(ctx, rep.TargetID)
	case model.ReportReview:
		rev, err := r.store.GetReview(ctx, rep.TargetID)
		if err != nil {
			return err
		}
		rev.Hidden = true
		_, err = r.store.UpdateReview(ctx, rev)
		return err
	default:
		return nil
	}
}
