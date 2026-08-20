package store

import (
	"context"
	"os"
	"path/filepath"
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
