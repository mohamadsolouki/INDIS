module github.com/IranProsperityProject/INDIS/services/identity

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/api/gen/go v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/blockchain v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/did v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/events v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/tls v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.6.0
	google.golang.org/grpc v1.64.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace (
	github.com/IranProsperityProject/INDIS/api/gen/go => ../../api/gen/go
	github.com/IranProsperityProject/INDIS/pkg/blockchain => ../../pkg/blockchain
	github.com/IranProsperityProject/INDIS/pkg/crypto => ../../pkg/crypto
	github.com/IranProsperityProject/INDIS/pkg/did => ../../pkg/did
	github.com/IranProsperityProject/INDIS/pkg/events => ../../pkg/events
	github.com/IranProsperityProject/INDIS/pkg/tls => ../../pkg/tls
)
