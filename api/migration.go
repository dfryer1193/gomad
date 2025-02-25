package api

import "time"

type MigrationCommonFields struct {
	Namespace string    `json:"namespace" db:"namespace"`
	User      string    `json:"user" db:"user"`
	Comment   string    `json:"comment" db:"comment"`
	DDL       string    `json:"ddl" db:"ddl"`
	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
}

type MigrationProto struct {
	MigrationCommonFields
	ShouldSkip bool   `db:"shouldSkip""`
	Signature  uint64 `db:"id"`
}

// Migration represents a database migration record
type Migration struct {
	MigrationCommonFields
	ID          uint64    `json:"id" db:"id"`
	CompletedAt time.Time `json:"completedAt" db:"completedAt"`
}
