package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type TaskService struct {
	store  store.Store
	notify *NotifyService
	credit *CreditHelper
	clock  Clock
}

func NewTaskService(s store.Store, notify *NotifyService, credit *CreditHelper, clock Clock) *TaskService {
	return &TaskService{store: s, notify: notify, credit: credit, clock: clock}
}

func (t *TaskService) Get(ctx context.Context, actor model.User, id string) (model.TaskView, error) {
	task, err := t.store.GetTask(ctx, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if !task.IsParty(actor.ID) && !actor.IsAdmin() {
		return model.TaskView{}, model.ErrNotTaskParty
	}
	return t.view(ctx, task)
}

func (t *TaskService) Mine(ctx context.Context, actor model.User) ([]model.TaskView, error) {
	tasks, err := t.store.ListTasksByUser(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]model.TaskView, 0, len(tasks))
	for _, task := range tasks {
		v, err := t.view(ctx, task)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

func (t *TaskService) ConfirmStart(ctx context.Context, actor model.User, id string) (model.TaskView, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.TaskView{}, err
	}
	// 在同一把写锁内完成“设置己方确认 + 双方都确认后推进到 in_progress +
	// 同步帖子状态”，避免两个参与方并发确认时各自基于旧副本回写而互相覆盖。
	updated, activated, err := t.store.ConfirmTaskStart(ctx, id, actor.ID, actor.IsAdmin())
	if err != nil {
		return model.TaskView{}, err
	}
	if activated {
		t.notify.Push(ctx, updated.RequesterID, model.NotifyTaskStarted, "互助已开始", "双方已确认开始。", updated.ID, "task")
		t.notify.Push(ctx, updated.HelperID, model.NotifyTaskStarted, "互助已开始", "双方已确认开始。", updated.ID, "task")
	}
	return t.view(ctx, updated)
}

func (t *TaskService) MarkComplete(ctx context.Context, actor model.User, id string) (model.TaskView, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.TaskView{}, err
	}
	task, err := t.mustParty(ctx, actor, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if !actor.IsAdmin() && actor.ID != task.HelperID {
		return model.TaskView{}, model.ErrNotHelper
	}
	if task.Status != model.TaskInProgress {
		return model.TaskView{}, model.ErrInvalidTaskStatus
	}
	task.Status = model.TaskPendingConfirm
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		return model.TaskView{}, err
	}
	if err := t.syncPostStatus(ctx, updated, model.PostPendingConfirm); err != nil {
		// 帖子状态持久化失败：把任务回退回 in_progress，使任务与帖子保持一致且可重试。
		// 否则任务停留在 pending_confirm 而帖子仍是 in_progress，再次标记会被状态校验拒绝，死锁。
		task.Status = model.TaskInProgress
		_, _ = t.store.UpdateTask(ctx, task)
		return model.TaskView{}, err
	}
	t.notify.Push(ctx, task.RequesterID, model.NotifyTaskCompleted, "帮助方已标记完成", "请确认本次互助是否完成。", task.ID, "task")
	return t.view(ctx, updated)
}

func (t *TaskService) ConfirmComplete(ctx context.Context, actor model.User, id string) (model.TaskView, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.TaskView{}, err
	}
	task, err := t.mustParty(ctx, actor, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if !actor.IsAdmin() && actor.ID != task.RequesterID {
		return model.TaskView{}, model.ErrNotRequester
	}
	if task.Status != model.TaskPendingConfirm {
		return model.TaskView{}, model.ErrInvalidTaskStatus
	}

	// 先写入两笔信用分（最易失败的一步），都成功后再提交任务/帖子终态。
	// 任一步失败时按反序冲销已落账的信用分，使任务维持 pending_confirm，保证可安全重试。
	undoHelper := func() {
		_, _ = t.credit.Apply(ctx, task.HelperID, -2, model.CreditCompleteHelper, task.ID, "完成帮助（回滚）")
	}
	undoBoth := func() {
		_, _ = t.credit.Apply(ctx, task.RequesterID, -1, model.CreditCompleteRequester, task.ID, "完成求助（回滚）")
		undoHelper()
	}

	if _, err := t.credit.Apply(ctx, task.HelperID, 2, model.CreditCompleteHelper, task.ID, "完成帮助"); err != nil {
		return model.TaskView{}, err
	}
	if _, err := t.credit.Apply(ctx, task.RequesterID, 1, model.CreditCompleteRequester, task.ID, "完成求助"); err != nil {
		undoHelper()
		return model.TaskView{}, err
	}

	now := t.clock.Now()
	task.Status = model.TaskCompleted
	task.CompleteAt = &now
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		undoBoth()
		return model.TaskView{}, err
	}
	if err := t.syncPostStatus(ctx, updated, model.PostCompleted); err != nil {
		undoBoth()
		task.Status = model.TaskPendingConfirm
		task.CompleteAt = nil
		_, _ = t.store.UpdateTask(ctx, task)
		return model.TaskView{}, err
	}
	t.bumpCounts(ctx, task)
	t.notify.Push(ctx, task.HelperID, model.NotifyTaskCompleted, "互助已完成", "对方已确认完成，可以评价。", task.ID, "task")
	t.notify.Push(ctx, task.RequesterID, model.NotifyTaskCompleted, "互助已完成", "可以评价对方。", task.ID, "task")
	return t.view(ctx, updated)
}

func (t *TaskService) Dispute(ctx context.Context, actor model.User, id string, in model.DisputeInput) (model.TaskView, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.TaskView{}, err
	}
	task, err := t.mustParty(ctx, actor, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if !actor.IsAdmin() && actor.ID != task.RequesterID {
		return model.TaskView{}, model.ErrNotRequester
	}
	if task.Status != model.TaskPendingConfirm {
		return model.TaskView{}, model.ErrInvalidTaskStatus
	}
	task.Status = model.TaskDisputed
	task.DisputeReason = in.Reason
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		return model.TaskView{}, err
	}
	t.notify.Push(ctx, task.HelperID, model.NotifyTaskDisputed, "对方提出争议", in.Reason, task.ID, "task")
	return t.view(ctx, updated)
}

func (t *TaskService) Cancel(ctx context.Context, actor model.User, id string, in model.CancelInput) (model.TaskView, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.TaskView{}, err
	}
	task, err := t.mustParty(ctx, actor, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if task.Status.IsTerminal() {
		return model.TaskView{}, model.ErrInvalidTaskStatus
	}
	afterStart := task.Status == model.TaskInProgress || task.Status == model.TaskPendingConfirm || task.Status == model.TaskDisputed
	deduct := afterStart && !actor.IsAdmin()
	// 先写入开始后取消的信用处罚（最易失败的一步），成功后再提交任务/帖子终态。
	// 任一步后续失败时按反序冲销已落账的信用分，使任务维持原状态且可安全重试；
	// 否则任务/帖子已变 cancelled 终态而信用分未扣，再次取消会被上面的终态校验
	// 拒绝，处罚永远无法补记。
	undoDeduct := func() {
		_, _ = t.credit.Apply(ctx, actor.ID, 3, model.CreditCancelAfterStart, task.ID, "开始后取消（回滚）")
	}
	if deduct {
		if _, err := t.credit.Apply(ctx, actor.ID, -3, model.CreditCancelAfterStart, task.ID, "开始后取消"); err != nil {
			return model.TaskView{}, err
		}
	}
	prevStatus := task.Status
	prevReason := task.CancelReason
	prevCancelledBy := task.CancelledBy
	task.Status = model.TaskCancelled
	task.CancelReason = in.Reason
	task.CancelledBy = actor.ID
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		if deduct {
			undoDeduct()
		}
		return model.TaskView{}, err
	}
	if err := t.syncPostStatus(ctx, updated, model.PostCancelled); err != nil {
		if deduct {
			undoDeduct()
		}
		task.Status = prevStatus
		task.CancelReason = prevReason
		task.CancelledBy = prevCancelledBy
		_, _ = t.store.UpdateTask(ctx, task)
		return model.TaskView{}, err
	}
	other := task.Counterpart(actor.ID)
	t.notify.Push(ctx, other, model.NotifyTaskCancelled, "互助已取消", in.Reason, task.ID, "task")
	return t.view(ctx, updated)
}

func (t *TaskService) mustParty(ctx context.Context, actor model.User, id string) (model.Task, error) {
	task, err := t.store.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if !task.IsParty(actor.ID) && !actor.IsAdmin() {
		return model.Task{}, model.ErrNotTaskParty
	}
	return task, nil
}

func (t *TaskService) syncPostStatus(ctx context.Context, task model.Task, status model.PostStatus) error {
	post, err := t.store.GetPost(ctx, task.PostID)
	if err != nil {
		return err
	}
	post.Status = status
	if status == model.PostCancelled {
		post.ClosedReason = task.CancelReason
	}
	_, err = t.store.UpdatePost(ctx, post)
	return err
}

func (t *TaskService) bumpCounts(ctx context.Context, task model.Task) {
	if helper, err := t.store.GetUserByID(ctx, task.HelperID); err == nil {
		helper.HelpCount++
		_, _ = t.store.UpdateUser(ctx, helper)
	}
	if req, err := t.store.GetUserByID(ctx, task.RequesterID); err == nil {
		req.RequestCount++
		_, _ = t.store.UpdateUser(ctx, req)
	}
}

func (t *TaskService) view(ctx context.Context, task model.Task) (model.TaskView, error) {
	post, err := t.store.GetPost(ctx, task.PostID)
	if err != nil {
		return model.TaskView{}, err
	}
	req, err := publicOf(ctx, t.store, task.RequesterID)
	if err != nil {
		return model.TaskView{}, err
	}
	help, err := publicOf(ctx, t.store, task.HelperID)
	if err != nil {
		return model.TaskView{}, err
	}
	return model.TaskView{Task: task, Post: post, Requester: req, Helper: help}, nil
}
