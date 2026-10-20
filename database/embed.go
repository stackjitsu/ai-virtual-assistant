package database

import "embed"

// Migrations contains versioned, SQLite-compatible schema statements.
//
//go:embed migrations/*.sql
var Migrations embed.FS
