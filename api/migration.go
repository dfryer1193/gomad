package api

import "time"

// Migration represents a database migration record
type Migration struct {
	ID        int       `db:"id" json:"id"`
	Namespace string    `db:"namespace" json:"namespace"`
	User      string    `db:"user" json:"user"`
	Comment   string    `db:"comment" json:"comment"`
	DDL       string    `db:"ddl" json:"ddl"`
	Completed time.Time `db:"completed" json:"completed"`
}
