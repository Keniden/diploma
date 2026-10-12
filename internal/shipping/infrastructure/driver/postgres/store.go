package pgdriver

import "last-mile-task/internal/platform/database"

type Store struct {
	*database.Repository
}

func New(repo *database.Repository) *Store {
	return &Store{
		Repository: repo,
	}
}
