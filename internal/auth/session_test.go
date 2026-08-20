package auth

import "go02-neighbor-help/internal/model"

func dummyUser(id string) model.User {
	return model.User{ID: id, Username: "n", Role: model.RoleResident}
}
