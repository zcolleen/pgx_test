module github.com/zcolleen/pgx_test

go 1.21.4

require github.com/jackc/pgx/v5 v5.5.1

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.9.0 // indirect
)

replace github.com/jackc/pgx/v5 => github.com/zcolleen/pgx/v5 v5.0.0-20231220223345-731a8c2a6611
