package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
)

func SeedAdmin(ctx context.Context, s *MemoryStore, hasher *auth.PasswordHasher, username, password string) error {
	if username == "" {
		return fmt.Errorf("seed: empty admin username")
	}
	if password == "" {
		return fmt.Errorf("seed: empty admin password")
	}
	users, err := s.ListUsers(ctx, model.UserFilter{Role: model.RoleAdmin})
	if err != nil {
		return fmt.Errorf("seed: list admins: %w", err)
	}
	if len(users) > 0 {
		return nil
	}
	salt, hash, iterations, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("seed: hash password: %w", err)
	}
	u, err := s.CreateUser(ctx, model.User{
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   iterations,
		Role:         model.RoleAdmin,
		Status:       model.UserActive,
		DisplayName:  "系统管理员",
		CreditScore:  model.CreditInitial,
	})
	if err != nil {
		return fmt.Errorf("seed: create admin: %w", err)
	}
	log.Printf("seed: created admin user %q (id=%s)", u.Username, u.ID)
	return nil
}

func SeedDemo(ctx context.Context, s *MemoryStore, hasher *auth.PasswordHasher) error {
	residents, err := s.ListUsers(ctx, model.UserFilter{Role: model.RoleResident})
	if err != nil {
		return err
	}
	if len(residents) > 0 {
		return nil
	}

	alice, err := seedResident(ctx, s, hasher, "alice", "alice123", "Alice 邻里", "2", "1", "201")
	if err != nil {
		return err
	}
	bob, err := seedResident(ctx, s, hasher, "bob", "bob123", "Bob 热心", "3", "2", "502")
	if err != nil {
		return err
	}

	end := time.Now().Add(48 * time.Hour)
	p1, err := s.CreatePost(ctx, model.HelpPost{
		Type:          model.PostRequest,
		Status:        model.PostOpen,
		Category:      model.CategoryDelivery,
		Urgency:       model.UrgencyNormal,
		Title:         "帮我代取快递",
		Content:       "下午 3 点前帮取 2 号楼门口菜鸟驿站两个小件，放门口即可。",
		Building:      "2",
		LocationNote:  "2 号楼一单元门口",
		TimeWindowEnd: &end,
		RewardType:    model.RewardThanks,
		AuthorID:      alice.ID,
	})
	if err != nil {
		return err
	}
	_, err = s.CreatePost(ctx, model.HelpPost{
		Type:         model.PostOffer,
		Status:       model.PostOpen,
		Category:     model.CategoryGrocery,
		Urgency:      model.UrgencyLow,
		Title:        "周末可以帮忙买菜",
		Content:      "周六上午去菜市场，同栋邻居需要代买可以留言，蔬菜水果都可以。",
		Building:     "3",
		RewardType:   model.RewardNone,
		AuthorID:     bob.ID,
	})
	if err != nil {
		return err
	}
	log.Printf("seed: demo residents alice/bob and sample posts (first post id=%s)", p1.ID)
	return nil
}

func seedResident(ctx context.Context, s *MemoryStore, hasher *auth.PasswordHasher, username, password, display, building, unit, room string) (model.User, error) {
	salt, hash, iterations, err := hasher.Hash(password)
	if err != nil {
		return model.User{}, err
	}
	return s.CreateUser(ctx, model.User{
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   iterations,
		Role:         model.RoleResident,
		Status:       model.UserActive,
		DisplayName:  display,
		Building:     building,
		Unit:         unit,
		Room:         room,
		CreditScore:  model.CreditInitial,
		Bio:          "小区邻居",
	})
}
