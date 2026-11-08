// Package migrations embeds the golang-migrate SQL files so the binary can
// migrate its own database. Files are append-only once applied to production.
package migrations

import "embed"

// FS holds every NNNNNN_name.{up,down}.sql file in this directory.
//
//go:embed *.sql
var FS embed.FS
