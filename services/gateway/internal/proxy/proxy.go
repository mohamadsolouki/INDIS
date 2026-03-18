// Package proxy manages gRPC client connections to all backend services.
package proxy

import (
	"fmt"

	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	biometricv1 "github.com/IranProsperityProject/INDIS/api/gen/go/biometric/v1"
	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	notificationv1 "github.com/IranProsperityProject/INDIS/api/gen/go/notification/v1"
	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients holds gRPC client stubs for all backend services.
type Clients struct {
	Identity     identityv1.IdentityServiceClient
	Credential   credentialv1.CredentialServiceClient
	Enrollment   enrollmentv1.EnrollmentServiceClient
	Biometric    biometricv1.BiometricServiceClient
	Audit        auditv1.AuditServiceClient
	Notification notificationv1.NotificationServiceClient
	Electoral    electoralv1.ElectoralServiceClient
	Justice      justicev1.JusticeServiceClient

	conns []*grpc.ClientConn
}

// New dials all backend services and returns a Clients bundle.
// Call Close() when done.
func New(identityAddr, credentialAddr, enrollmentAddr, biometricAddr,
	auditAddr, notificationAddr, electoralAddr, justiceAddr string) (*Clients, error) {

	dial := func(addr string) (*grpc.ClientConn, error) {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr, err)
		}
		return conn, nil
	}

	addrs := []string{
		identityAddr, credentialAddr, enrollmentAddr, biometricAddr,
		auditAddr, notificationAddr, electoralAddr, justiceAddr,
	}
	conns := make([]*grpc.ClientConn, 0, len(addrs))
	for _, addr := range addrs {
		conn, err := dial(addr)
		if err != nil {
			// Close already-opened connections before returning.
			for _, c := range conns {
				c.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
	}

	return &Clients{
		Identity:     identityv1.NewIdentityServiceClient(conns[0]),
		Credential:   credentialv1.NewCredentialServiceClient(conns[1]),
		Enrollment:   enrollmentv1.NewEnrollmentServiceClient(conns[2]),
		Biometric:    biometricv1.NewBiometricServiceClient(conns[3]),
		Audit:        auditv1.NewAuditServiceClient(conns[4]),
		Notification: notificationv1.NewNotificationServiceClient(conns[5]),
		Electoral:    electoralv1.NewElectoralServiceClient(conns[6]),
		Justice:      justicev1.NewJusticeServiceClient(conns[7]),
		conns:        conns,
	}, nil
}

// Close releases all underlying gRPC connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}
