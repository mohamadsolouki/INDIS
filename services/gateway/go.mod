module github.com/IranProsperityProject/INDIS/services/gateway

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/api/gen/go v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/metrics v0.0.0-00010101000000-000000000000
	github.com/IranProsperityProject/INDIS/pkg/tls v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.64.0
)

require (
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace (
	github.com/IranProsperityProject/INDIS/api/gen/go => ../../api/gen/go
	github.com/IranProsperityProject/INDIS/pkg/metrics => ../../pkg/metrics
	github.com/IranProsperityProject/INDIS/pkg/tls => ../../pkg/tls
)
