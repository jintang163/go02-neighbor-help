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
	task, err := t.mustParty(ctx, actor, id)
	if err != nil {
		return model.TaskView{}, err
	}
	if task.Status != model.TaskPendingStart {
		return model.TaskView{}, model.ErrInvalidTaskStatus
	}
	switch actor.ID {
	case task.RequesterID:
		task.RequesterStarted = true
	case task.HelperID:
		task.HelperStarted = true
	}
	if actor.IsAdmin() {
		task.RequesterStarted = true
		task.HelperStarted = true
	}
	if task.RequesterStarted && task.HelperStarted {
		now := t.clock.Now()
		task.Status = model.TaskInProgress
		task.StartAt = &now
		if err := t.syncPostStatus(ctx, task, model.PostInProgress); err != nil {
			return model.TaskView{}, err
		}
		t.notify.Push(ctx, task.RequesterID, model.NotifyTaskStarted, "互助已开始", "双方已确认开始。", task.ID, "task")
		t.notify.Push(ctx, task.HelperID, model.NotifyTaskStarted, "互助已开始", "双方已确认开始。", task.ID, "task")
	}
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		return model.TaskView{}, err
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
	now := t.clock.Now()
	task.Status = model.TaskCompleted
	task.CompleteAt = &now
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		return model.TaskView{}, err
	}
	if err := t.syncPostStatus(ctx, updated, model.PostCompleted); err != nil {
		return model.TaskView{}, err
	}
	if _, err := t.credit.Apply(ctx, task.HelperID, 2, model.CreditCompleteHelper, task.ID, "完成帮助"); err != nil {
		return model.TaskView{}, err
	}
	if _, err := t.credit.Apply(ctx, task.RequesterID, 1, model.CreditCompleteRequester, task.ID, "完成求助"); err != nil {
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
	task.Status = model.TaskCancelled
	task.CancelReason = in.Reason
	task.CancelledBy = actor.ID
	updated, err := t.store.UpdateTask(ctx, task)
	if err != nil {
		return model.TaskView{}, err
	}
	if err := t.syncPostStatus(ctx, updated, model.PostCancelled); err != nil {
		return model.TaskView{}, err
	}
	if afterStart && !actor.IsAdmin() {
		_, _ = t.credit.Apply(ctx, actor.ID, -3, model.CreditCancelAfterStart, task.ID, "开始后取消")
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
