package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/service"
	"go02-neighbor-help/internal/store"
)

type synchronizedTaskStore struct {
	store.Store
	mu      sync.Mutex
	entered int
	allIn   chan struct{}
}

var (
	errRequesterCreditWrite = errors.New("credit store write failed for requester: durable log unavailable")
	errHelperCreditWrite    = errors.New("credit store write failed for helper: durable log unavailable")
	errPostStatusWrite      = errors.New("post store write failed during mark complete: disk sync unavailable")
)

type failPostUpdateStore struct{ store.Store }

func (s *failPostUpdateStore) UpdatePost(ctx context.Context, post model.HelpPost) (model.HelpPost, error) {
	return model.HelpPost{}, errPostStatusWrite
}

// failFirstCreditStore 让 ApplyCredit 第一次调用即失败，用于验证帮助方信用写入
// 失败时不会留下任何部分提交（此时尚未推进任务/帖子终态，也无需冲销）。
type failFirstCreditStore struct {
	store.Store
	mu    sync.Mutex
	calls int
}

func (s *failFirstCreditStore) ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string) (model.User, model.CreditLog, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if call == 1 {
		return model.User{}, model.CreditLog{}, errHelperCreditWrite
	}
	return s.Store.ApplyCredit(ctx, userID, delta, reason, relatedID, note)
}

type failSecondCreditStore struct {
	store.Store
	mu    sync.Mutex
	calls int
}

func (s *failSecondCreditStore) ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string) (model.User, model.CreditLog, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if call == 2 {
		return model.User{}, model.CreditLog{}, errRequesterCreditWrite
	}
	return s.Store.ApplyCredit(ctx, userID, delta, reason, relatedID, note)
}

func newSynchronizedTaskStore(base store.Store) *synchronizedTaskStore {
	return &synchronizedTaskStore{Store: base, allIn: make(chan struct{})}
}

// ConfirmTaskStart 是 ConfirmStart 的真正临界区入口。这里用屏障制造并发：
// 两个参与方都进入后才开始各自调用底层 store，从而真正触发“先后进入、
// 各自回写”的竞争窗口，用来验证 MemoryStore.ConfirmTaskStart 的原子性。
func (s *synchronizedTaskStore) ConfirmTaskStart(ctx context.Context, taskID, actorID string, asAdmin bool) (model.Task, bool, error) {
	s.mu.Lock()
	s.entered++
	if s.entered == 2 {
		close(s.allIn)
	}
	s.mu.Unlock()
	select {
	case <-s.allIn:
	case <-ctx.Done():
		return model.Task{}, false, ctx.Err()
	}
	return s.Store.ConfirmTaskStart(ctx, taskID, actorID, asAdmin)
}

func setup(t *testing.T) (context.Context, *service.Services, *store.MemoryStore) {
	t.Helper()
	ctx := context.Background()
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(mem, hasher, sessions, nil, 10)
	return ctx, svc, mem
}

func mustUser(t *testing.T, ctx context.Context, svc *service.Services, name string) model.User {
	t.Helper()
	out, err := svc.Auth.Register(ctx, model.UserInput{
		Username: name, Password: name + "1234", DisplayName: name, Role: model.RoleResident,
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := svc.User.GetByID(ctx, out.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestMatchAndMutualReview(t *testing.T) {
	ctx, svc, _ := setup(t)
	alice := mustUser(t, ctx, svc, "alice")
	bob := mustUser(t, ctx, svc, "bob")

	post, err := svc.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryDelivery, Urgency: model.UrgencyNormal,
		Title: "帮取快递", Content: "2 号楼门口两个包裹", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.Status != model.PostOpen {
		t.Fatalf("want open, got %s", post.Status)
	}

	if _, err := svc.Match.Apply(ctx, alice, post.ID, model.ApplyInput{}); err == nil {
		t.Fatal("should not apply to own post")
	}

	app, err := svc.Match.Apply(ctx, bob, post.ID, model.ApplyInput{Message: "我去取"})
	if err != nil {
		t.Fatal(err)
	}

	tv, err := svc.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tv.HelperID != bob.ID || tv.RequesterID != alice.ID {
		t.Fatalf("roles mismatch: %+v", tv.Task)
	}

	if _, err := svc.Task.ConfirmStart(ctx, alice, tv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Task.ConfirmStart(ctx, bob, tv.ID); err != nil {
		t.Fatal(err)
	}
	started, err := svc.Task.Get(ctx, alice, tv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != model.TaskInProgress {
		t.Fatalf("want in_progress, got %s", started.Status)
	}

	if _, err := svc.Task.MarkComplete(ctx, alice, tv.ID); err == nil {
		t.Fatal("requester should not mark complete")
	}
	if _, err := svc.Task.MarkComplete(ctx, bob, tv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Task.ConfirmComplete(ctx, alice, tv.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Review.Submit(ctx, alice, tv.ID, model.ReviewInput{Score: 5, Tags: []string{"kind"}, Comment: "很热心"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review.Submit(ctx, alice, tv.ID, model.ReviewInput{Score: 5, Tags: []string{"kind"}}); err == nil {
		t.Fatal("duplicate review should fail")
	}
	if _, err := svc.Review.Submit(ctx, bob, tv.ID, model.ReviewInput{Score: 4, Tags: []string{"punctual"}}); err != nil {
		t.Fatal(err)
	}

	bob2, _ := svc.User.GetByID(ctx, bob.ID)
	if bob2.CreditScore <= model.CreditInitial {
		t.Fatalf("helper credit should increase, got %d", bob2.CreditScore)
	}
	if bob2.HelpCount != 1 {
		t.Fatalf("help count=%d", bob2.HelpCount)
	}
}

func TestUrgentRequiresCredit(t *testing.T) {
	ctx, svc, mem := setup(t)
	alice := mustUser(t, ctx, svc, "alice")
	alice.CreditScore = 30
	if _, err := mem.UpdateUser(ctx, alice); err != nil {
		t.Fatal(err)
	}
	alice, _ = svc.User.GetByID(ctx, alice.ID)
	_, err := svc.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryOther, Urgency: model.UrgencyUrgent,
		Title: "紧急求助", Content: "需要马上有人帮忙", Publish: true,
	})
	if err != model.ErrCreditRestricted {
		t.Fatalf("want credit restricted, got %v", err)
	}
}

func TestOfferMatchRoles(t *testing.T) {
	ctx, svc, _ := setup(t)
	alice := mustUser(t, ctx, svc, "alice")
	bob := mustUser(t, ctx, svc, "bob")
	post, err := svc.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostOffer, Category: model.CategoryGrocery, Urgency: model.UrgencyLow,
		Title: "周末可买菜", Content: "同栋可代买", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := svc.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	if err != nil {
		t.Fatal(err)
	}
	tv, err := svc.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tv.HelperID != alice.ID || tv.RequesterID != bob.ID {
		t.Fatalf("offer post should keep author as helper: %+v", tv.Task)
	}
}

func TestLoginRejectsBadPassword(t *testing.T) {
	ctx, svc, _ := setup(t)
	_ = mustUser(t, ctx, svc, "dave")
	_, err := svc.Auth.Login(ctx, model.LoginInput{Username: "dave", Password: "nope"})
	if err != model.ErrInvalidCredentials {
		t.Fatalf("got %v", err)
	}
}

func TestReviewTagMismatch(t *testing.T) {
	in := model.ReviewInput{Score: 5, Tags: []string{"late"}}
	if err := in.Validate(); err != model.ErrInvalidReviewTags {
		t.Fatalf("got %v", err)
	}
}

func TestCancelAfterStartDeductsCredit(t *testing.T) {
	ctx, svc, _ := setup(t)
	alice := mustUser(t, ctx, svc, "alice")
	bob := mustUser(t, ctx, svc, "bob")
	post, _ := svc.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryPet, Urgency: model.UrgencyNormal,
		Title: "照看猫咪", Content: "出门半天", Publish: true,
	})
	app, _ := svc.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	tv, _ := svc.Match.Accept(ctx, alice, app.ID)
	_, _ = svc.Task.ConfirmStart(ctx, alice, tv.ID)
	_, _ = svc.Task.ConfirmStart(ctx, bob, tv.ID)
	before, _ := svc.User.GetByID(ctx, bob.ID)
	if _, err := svc.Task.Cancel(ctx, bob, tv.ID, model.CancelInput{Reason: "临时有事"}); err != nil {
		t.Fatal(err)
	}
	after, _ := svc.User.GetByID(ctx, bob.ID)
	if after.CreditScore != before.CreditScore-3 {
		t.Fatalf("credit %d -> %d", before.CreditScore, after.CreditScore)
	}
}

func TestConcurrentConfirmStartActivatesTask(t *testing.T) {
	ctx := context.Background()
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	setupServices := service.NewServices(mem, hasher, sessions, nil, 10)
	alice := mustUser(t, ctx, setupServices, "alice")
	bob := mustUser(t, ctx, setupServices, "bob")

	post, err := setupServices.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryDelivery, Urgency: model.UrgencyNormal,
		Title: "并发确认开始", Content: "双方同时确认后应立即开始", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := setupServices.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	if err != nil {
		t.Fatal(err)
	}
	task, err := setupServices.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}

	coordinated := newSynchronizedTaskStore(mem)
	concurrentServices := service.NewServices(coordinated, hasher, sessions, nil, 10)
	results := make(chan error, 2)
	go func() {
		_, err := concurrentServices.Task.ConfirmStart(ctx, alice, task.ID)
		results <- err
	}()
	go func() {
		_, err := concurrentServices.Task.ConfirmStart(ctx, bob, task.ID)
		results <- err
	}()
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatalf("confirm start failed: %v", err)
		}
	}

	stored, err := mem.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.TaskInProgress || !stored.RequesterStarted || !stored.HelperStarted {
		t.Fatalf("both confirmations succeeded but task was not activated: status=%s requester_started=%t helper_started=%t", stored.Status, stored.RequesterStarted, stored.HelperStarted)
	}
}

func TestConfirmCompleteFailureKeepsWorkflowRetryable(t *testing.T) {
	ctx, setupServices, mem := setup(t)
	alice := mustUser(t, ctx, setupServices, "alice")
	bob := mustUser(t, ctx, setupServices, "bob")
	post, err := setupServices.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryDelivery, Urgency: model.UrgencyNormal,
		Title: "确认完成失败恢复", Content: "信用记录失败后仍应可以重试", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := setupServices.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	if err != nil {
		t.Fatal(err)
	}
	task, err := setupServices.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setupServices.Task.ConfirmStart(ctx, alice, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := setupServices.Task.ConfirmStart(ctx, bob, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := setupServices.Task.MarkComplete(ctx, bob, task.ID); err != nil {
		t.Fatal(err)
	}

	aliceBefore, err := mem.GetUserByID(ctx, alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	bobBefore, err := mem.GetUserByID(ctx, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	failing := &failSecondCreditStore{Store: mem}
	failingServices := service.NewServices(failing, auth.NewPasswordHasher(), auth.NewSessionManager(time.Hour), nil, 10)
	_, completionErr := failingServices.Task.ConfirmComplete(ctx, alice, task.ID)
	if !errors.Is(completionErr, errRequesterCreditWrite) {
		t.Fatalf("want requester credit failure %q, got %v", errRequesterCreditWrite, completionErr)
	}

	storedTask, err := mem.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedPost, err := mem.GetPost(ctx, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	aliceAfter, err := mem.GetUserByID(ctx, alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	bobAfter, err := mem.GetUserByID(ctx, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedTask.Status != model.TaskPendingConfirm || storedPost.Status != model.PostPendingConfirm || aliceAfter.CreditScore != aliceBefore.CreditScore || bobAfter.CreditScore != bobBefore.CreditScore {
		t.Fatalf("failed completion left partial state: task=%s post=%s requester_credit=%d->%d helper_credit=%d->%d error=%v", storedTask.Status, storedPost.Status, aliceBefore.CreditScore, aliceAfter.CreditScore, bobBefore.CreditScore, bobAfter.CreditScore, completionErr)
	}

	// 失败后用正常 store 重试，应能顺利完成：任务/帖子变 completed，双方信用分各加一次。
	retry, err := setupServices.Task.ConfirmComplete(ctx, alice, task.ID)
	if err != nil {
		t.Fatalf("retry confirm complete failed: %v", err)
	}
	if retry.Status != model.TaskCompleted {
		t.Fatalf("retry want completed, got %s", retry.Status)
	}
	bobFinal, _ := setupServices.User.GetByID(ctx, bob.ID)
	aliceFinal, _ := setupServices.User.GetByID(ctx, alice.ID)
	if bobFinal.CreditScore != bobBefore.CreditScore+2 {
		t.Fatalf("retry helper credit want %d, got %d", bobBefore.CreditScore+2, bobFinal.CreditScore)
	}
	if aliceFinal.CreditScore != aliceBefore.CreditScore+1 {
		t.Fatalf("retry requester credit want %d, got %d", aliceBefore.CreditScore+1, aliceFinal.CreditScore)
	}
}

func TestConfirmCompleteHelperCreditFailureKeepsWorkflowRetryable(t *testing.T) {
	ctx, setupServices, mem := setup(t)
	alice := mustUser(t, ctx, setupServices, "alice")
	bob := mustUser(t, ctx, setupServices, "bob")
	post, err := setupServices.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryDelivery, Urgency: model.UrgencyNormal,
		Title: "帮助方信用失败恢复", Content: "帮助方信用记录失败后仍应可以重试", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := setupServices.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	if err != nil {
		t.Fatal(err)
	}
	task, err := setupServices.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = setupServices.Task.ConfirmStart(ctx, alice, task.ID)
	_, _ = setupServices.Task.ConfirmStart(ctx, bob, task.ID)
	_, _ = setupServices.Task.MarkComplete(ctx, bob, task.ID)

	aliceBefore, _ := mem.GetUserByID(ctx, alice.ID)
	bobBefore, _ := mem.GetUserByID(ctx, bob.ID)

	// 第一次 ApplyCredit 即失败：帮助方信用尚未落账，不应有部分提交，也无需冲销。
	failing := &failFirstCreditStore{Store: mem}
	failingServices := service.NewServices(failing, auth.NewPasswordHasher(), auth.NewSessionManager(time.Hour), nil, 10)
	_, completionErr := failingServices.Task.ConfirmComplete(ctx, alice, task.ID)
	if !errors.Is(completionErr, errHelperCreditWrite) {
		t.Fatalf("want helper credit failure %q, got %v", errHelperCreditWrite, completionErr)
	}

	storedTask, _ := mem.GetTask(ctx, task.ID)
	storedPost, _ := mem.GetPost(ctx, post.ID)
	aliceAfter, _ := mem.GetUserByID(ctx, alice.ID)
	bobAfter, _ := mem.GetUserByID(ctx, bob.ID)
	if storedTask.Status != model.TaskPendingConfirm || storedPost.Status != model.PostPendingConfirm || aliceAfter.CreditScore != aliceBefore.CreditScore || bobAfter.CreditScore != bobBefore.CreditScore {
		t.Fatalf("helper-credit failure left partial state: task=%s post=%s requester_credit=%d->%d helper_credit=%d->%d error=%v", storedTask.Status, storedPost.Status, aliceBefore.CreditScore, aliceAfter.CreditScore, bobBefore.CreditScore, bobAfter.CreditScore, completionErr)
	}
}

func TestMarkCompleteFailureKeepsTaskRetryable(t *testing.T) {
	ctx, setupServices, mem := setup(t)
	alice := mustUser(t, ctx, setupServices, "alice")
	bob := mustUser(t, ctx, setupServices, "bob")
	post, err := setupServices.Post.Create(ctx, alice, model.PostInput{
		Type: model.PostRequest, Category: model.CategoryDelivery, Urgency: model.UrgencyNormal,
		Title: "标记完成失败恢复", Content: "帖子状态失败时任务仍应可以重试", Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := setupServices.Match.Apply(ctx, bob, post.ID, model.ApplyInput{})
	if err != nil {
		t.Fatal(err)
	}
	task, err := setupServices.Match.Accept(ctx, alice, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setupServices.Task.ConfirmStart(ctx, alice, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := setupServices.Task.ConfirmStart(ctx, bob, task.ID); err != nil {
		t.Fatal(err)
	}

	failingServices := service.NewServices(&failPostUpdateStore{Store: mem}, auth.NewPasswordHasher(), auth.NewSessionManager(time.Hour), nil, 10)
	_, markErr := failingServices.Task.MarkComplete(ctx, bob, task.ID)
	if !errors.Is(markErr, errPostStatusWrite) {
		t.Fatalf("want post status failure %q, got %v", errPostStatusWrite, markErr)
	}
	storedTask, err := mem.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedPost, err := mem.GetPost(ctx, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedTask.Status != model.TaskInProgress || storedPost.Status != model.PostInProgress {
		t.Fatalf("failed mark complete left workflow inconsistent: task=%s post=%s error=%v", storedTask.Status, storedPost.Status, markErr)
	}
}
