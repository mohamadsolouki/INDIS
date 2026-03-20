module github.com/IranProsperityProject/INDIS/tools/pqc-migrate

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/pkg/crypto v0.0.0
	github.com/IranProsperityProject/INDIS/pkg/vc v0.0.0
	github.com/jackc/pgx/v5 v5.5.5
)

replace (
	github.com/IranProsperityProject/INDIS/pkg/crypto => ../../pkg/crypto
	github.com/IranProsperityProject/INDIS/pkg/vc => ../../pkg/vc
)
