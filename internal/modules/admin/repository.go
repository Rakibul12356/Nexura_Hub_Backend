package admin

import (
	"context"
	"database/sql"
)

type AdminRepository interface {
	Ping(ctx context.Context) error
}

type postgresAdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) AdminRepository {
	return &postgresAdminRepository{db: db}
}

func (r *postgresAdminRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
