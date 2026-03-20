module github.com/IranProsperityProject/INDIS/services/card

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/pkg/migrate v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.6.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/IranProsperityProject/INDIS/pkg/migrate => ../../pkg/migrate
replace github.com/IranProsperityProject/INDIS/pkg/tracing => ../../pkg/tracing
