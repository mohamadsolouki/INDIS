// Package proxy manages gRPC client connections to all backend services.
package proxy

import (
	"fmt"

	auditv1 "github.com/IranProsperityProject/INDIS/api/gen/go/audit/v1"
	biometricv1 "github.com/IranProsperityProject/INDIS/api/gen/go/biometric/v1"
	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	enrollmentv1 "github.com/IranProsperityProject/INDIS/api/gen/go/enrollment/v1"
	identityv1 "github.com/IranProsperityProject/INDIS/api/gen/go/identity/v1"
	justicev1 "github.com/IranProsperityProject/INDIS/api/gen/go/justice/v1"
	notificationv1 "github.com/IranProsperityProject/INDIS/api/gen/go/notification/v1"
	"github.com/IranProsperityProject/INDIS/services/gateway/internal/circuitbreaker"
	indistls "github.com/IranProsperityProject/INDIS/pkg/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TransportConfig controls how gateway dials backend gRPC services.
type TransportConfig struct {
	Mode           string
	CAFile         string
	ClientCertFile string
	ClientKeyFile  string
}

// Clients holds gRPC client stubs for all backend services.
// Each service has its own CircuitBreaker so that failures in one backend do
// not affect availability of others.
type Clients struct {
	Identity     identityv1.IdentityServiceClient
	Credential   credentialv1.CredentialServiceClient
	Enrollment   enrollmentv1.EnrollmentServiceClient
	Biometric    biometricv1.BiometricServiceClient
	Audit        auditv1.AuditServiceClient
	Notification notificationv1.NotificationServiceClient
	Electoral    electoralv1.ElectoralServiceClient
	Justice      justicev1.JusticeServiceClient

	// Circuit breakers — one per backend service.
	CBIdentity     *circuitbreaker.CircuitBreaker
	CBCredential   *circuitbreaker.CircuitBreaker
	CBEnrollment   *circuitbreaker.CircuitBreaker
	CBBiometric    *circuitbreaker.CircuitBreaker
	CBAudit        *circuitbreaker.CircuitBreaker
	CBNotification *circuitbreaker.CircuitBreaker
	CBElectoral    *circuitbreaker.CircuitBreaker
	CBJustice      *circuitbreaker.CircuitBreaker

	conns []*grpc.ClientConn
}

// New dials all backend services and returns a Clients bundle.
// Call Close() when done.
func New(identityAddr, credentialAddr, enrollmentAddr, biometricAddr,
	auditAddr, notificationAddr, electoralAddr, justiceAddr string, transportCfg TransportConfig) (*Clients, error) {

	var clientCreds credentials.TransportCredentials
	switch transportCfg.Mode {
	case "", "plaintext":
		clientCreds = insecure.NewCredentials()
	case "tls":
		var (
			creds credentials.TransportCredentials
			err   error
		)
		if transportCfg.ClientCertFile != "" {
			creds, err = indistls.LoadClientMTLS(transportCfg.CAFile, transportCfg.ClientCertFile, transportCfg.ClientKeyFile)
		} else {
			creds, err = indistls.LoadClientTLS(transportCfg.CAFile)
		}
		if err != nil {
			return nil, fmt.Errorf("load backend TLS credentials: %w", err)
		}
		clientCreds = creds
	case "tls_insecure_skip_verify":
		clientCreds = indistls.LoadClientTLSInsecureSkipVerify()
	default:
		return nil, fmt.Errorf("unsupported transport mode %q", transportCfg.Mode)
	}

	dial := func(addr string) (*grpc.ClientConn, error) {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(clientCreds))
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

		CBIdentity:     circuitbreaker.New(),
		CBCredential:   circuitbreaker.New(),
		CBEnrollment:   circuitbreaker.New(),
		CBBiometric:    circuitbreaker.New(),
		CBAudit:        circuitbreaker.New(),
		CBNotification: circuitbreaker.New(),
		CBElectoral:    circuitbreaker.New(),
		CBJustice:      circuitbreaker.New(),

		conns: conns,
	}, nil
}

// Close releases all underlying gRPC connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}
