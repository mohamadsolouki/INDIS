module github.com/IranProsperityProject/INDIS/services/credential

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/api/gen/go v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/blockchain v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/vc v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.6.0
	google.golang.org/grpc v1.64.0
)

replace (
	github.com/IranProsperityProject/INDIS/api/gen/go => ../../api/gen/go
	github.com/IranProsperityProject/INDIS/pkg/blockchain => ../../pkg/blockchain
	github.com/IranProsperityProject/INDIS/pkg/vc => ../../pkg/vc
)
