module github.com/IranProsperityProject/INDIS/services/gateway

go 1.22.0

require (
	github.com/IranProsperityProject/INDIS/api/gen/go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
)

replace (
	github.com/IranProsperityProject/INDIS/api/gen/go => ../../api/gen/go
)
