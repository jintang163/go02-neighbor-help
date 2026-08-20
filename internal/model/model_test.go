package model

import "testing"

func TestCreditLevelOf(t *testing.T) {
	cases := []struct {
		score int
		want  CreditLevel
	}{
		{0, CreditRestricted},
		{39, CreditRestricted},
		{40, CreditNew},
		{60, CreditNormal},
		{80, CreditTrusted},
		{95, CreditExcellent},
	}
	for _, c := range cases {
		if got := CreditLevelOf(c.score); got != c.want {
			t.Fatalf("score %d got %s want %s", c.score, got, c.want)
		}
	}
}

func TestPostVisibleTo(t *testing.T) {
	author := User{ID: "u1", Role: RoleResident}
	other := User{ID: "u2", Role: RoleResident}
	admin := User{ID: "a", Role: RoleAdmin}
	draft := HelpPost{AuthorID: "u1", Status: PostDraft}
	if draft.VisibleTo(other) {
		t.Fatal("other should not see draft")
	}
	if !draft.VisibleTo(author) || !draft.VisibleTo(admin) {
		t.Fatal("author/admin should see draft")
	}
}

func TestUsernameRules(t *testing.T) {
	if IsValidUsername("ab") || IsValidUsername("a-b") {
		t.Fatal("invalid accepted")
	}
	if !IsValidUsername("alice_1") {
		t.Fatal("valid rejected")
	}
}
