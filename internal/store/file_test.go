package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go02-neighbor-help/internal/model"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	u, err := fs.Store().CreateUser(ctx, model.User{
		Username: "a1", DisplayName: "A", Role: model.RoleResident, CreditScore: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs2.Store().GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "a1" {
		t.Fatalf("got %+v", got)
	}
}

func TestAcceptMatchRejectsOthers(t *testing.T) {
	mem := NewMemoryStore(time.Now, nil)
	ctx := context.Background()
	author, _ := mem.CreateUser(ctx, model.User{Username: "au", DisplayName: "au", Role: model.RoleResident})
	a1, _ := mem.CreateUser(ctx, model.User{Username: "a1", DisplayName: "a1", Role: model.RoleResident})
	a2, _ := mem.CreateUser(ctx, model.User{Username: "a2", DisplayName: "a2", Role: model.RoleResident})
	post, err := mem.CreatePost(ctx, model.HelpPost{
		Type: model.PostRequest, Status: model.PostOpen, Title: "t", Content: "c", AuthorID: author.ID,
		Category: model.CategoryOther, Urgency: model.UrgencyNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	app1, _ := mem.CreateApplication(ctx, model.Application{PostID: post.ID, ApplicantID: a1.ID})
	app2, _ := mem.CreateApplication(ctx, model.Application{PostID: post.ID, ApplicantID: a2.ID})
	_, task, p2, err := mem.AcceptMatch(ctx, app1.ID, author.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Status != model.PostMatched || task.HelperID != a1.ID {
		t.Fatalf("bad match %+v %+v", p2, task)
	}
	other, _ := mem.GetApplication(ctx, app2.ID)
	if other.Status != model.AppRejected {
		t.Fatalf("want rejected, got %s", other.Status)
	}
}

// TestConfirmTaskStartConcurrentAtomic 验证两个参与方几乎同时确认开始时，
// 两次确认不会互相覆盖，双方都确认后任务可靠进入 in_progress。
func TestConfirmTaskStartConcurrentAtomic(t *testing.T) {
	mem := NewMemoryStore(time.Now, nil)
	ctx := context.Background()
	requester, _ := mem.CreateUser(ctx, model.User{Username: "rq", DisplayName: "rq", Role: model.RoleResident})
	helper, _ := mem.CreateUser(ctx, model.User{Username: "hp", DisplayName: "hp", Role: model.RoleResident})
	post, err := mem.CreatePost(ctx, model.HelpPost{
		Type: model.PostRequest, Status: model.PostOpen, Title: "t", Content: "c", AuthorID: requester.ID,
		Category: model.CategoryOther, Urgency: model.UrgencyNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := mem.CreateApplication(ctx, model.Application{PostID: post.ID, ApplicantID: helper.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, task, _, err := mem.AcceptMatch(ctx, app.ID, requester.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	// 多轮并发，放大竞争窗口，配合 -race 复现覆盖问题。
	for i := 0; i < 50; i++ {
		// 每轮重置任务到 pending_start，重新制造并发确认。
		reset, _ := mem.GetTask(ctx, task.ID)
		reset.Status = model.TaskPendingStart
		reset.RequesterStarted = false
		reset.HelperStarted = false
		reset.StartAt = nil
		_, _ = mem.UpdateTask(ctx, reset)

		var wg sync.WaitGroup
		wg.Add(2)
		errs := make(chan error, 2)
		go func() {
			defer wg.Done()
			_, _, err := mem.ConfirmTaskStart(ctx, task.ID, requester.ID, false)
			errs <- err
		}()
		go func() {
			defer wg.Done()
			_, _, err := mem.ConfirmTaskStart(ctx, task.ID, helper.ID, false)
			errs <- err
		}()
		wg.Wait()
		close(errs)
		for e := range errs {
			if e != nil {
				t.Fatalf("round %d confirm failed: %v", i, e)
			}
		}

		stored, _ := mem.GetTask(ctx, task.ID)
		if stored.Status != model.TaskInProgress || !stored.RequesterStarted || !stored.HelperStarted {
			t.Fatalf("round %d: both confirmed but task not activated: status=%s requester=%t helper=%t",
				i, stored.Status, stored.RequesterStarted, stored.HelperStarted)
		}
	}
}

// TestConfirmTaskStartIdempotentAndStatusGuard 验证重复确认无副作用，
// 且任务离开 pending_start 后再确认返回状态错误。
func TestConfirmTaskStartIdempotentAndStatusGuard(t *testing.T) {
	mem := NewMemoryStore(time.Now, nil)
	ctx := context.Background()
	requester, _ := mem.CreateUser(ctx, model.User{Username: "rq2", DisplayName: "rq2", Role: model.RoleResident})
	helper, _ := mem.CreateUser(ctx, model.User{Username: "hp2", DisplayName: "hp2", Role: model.RoleResident})
	post, _ := mem.CreatePost(ctx, model.HelpPost{
		Type: model.PostRequest, Status: model.PostOpen, Title: "t", Content: "c", AuthorID: requester.ID,
		Category: model.CategoryOther, Urgency: model.UrgencyNormal,
	})
	app, _ := mem.CreateApplication(ctx, model.Application{PostID: post.ID, ApplicantID: helper.ID})
	_, task, _, _ := mem.AcceptMatch(ctx, app.ID, requester.ID, false)

	// 同一方重复确认不应推进状态。
	updated, activated, err := mem.ConfirmTaskStart(ctx, task.ID, requester.ID, false)
	if err != nil || activated {
		t.Fatalf("first confirm: err=%v activated=%t", err, activated)
	}
	if !updated.RequesterStarted || updated.HelperStarted {
		t.Fatalf("only requester should be started: requester=%t helper=%t", updated.RequesterStarted, updated.HelperStarted)
	}
	again, activated2, err := mem.ConfirmTaskStart(ctx, task.ID, requester.ID, false)
	if err != nil || activated2 || !again.RequesterStarted || again.HelperStarted {
		t.Fatalf("redundant requester confirm should be no-op: err=%v activated=%t", err, activated2)
	}

	// helper 确认后任务进入进行中。
	final, activated3, err := mem.ConfirmTaskStart(ctx, task.ID, helper.ID, false)
	if err != nil || !activated3 || final.Status != model.TaskInProgress {
		t.Fatalf("helper confirm should activate: err=%v activated=%t status=%s", err, activated3, final.Status)
	}

	// 任务已离开 pending_start，再次确认应拒绝。
	if _, _, err := mem.ConfirmTaskStart(ctx, task.ID, requester.ID, false); err != model.ErrInvalidTaskStatus {
		t.Fatalf("confirm after activation should fail with invalid status, got %v", err)
	}
}
