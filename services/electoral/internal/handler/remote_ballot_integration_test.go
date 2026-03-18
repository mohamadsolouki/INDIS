package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	electoralv1 "github.com/IranProsperityProject/INDIS/api/gen/go/electoral/v1"
	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/repository"
	"github.com/IranProsperityProject/INDIS/services/electoral/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func setupRemoteBallotIntegration(t *testing.T) (context.Context, electoralv1.ElectoralServiceClient, *pgxpool.Pool, func()) {
	t.Helper()

	dsn := os.Getenv("ELECTORAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ELECTORAL_TEST_DATABASE_URL to run gRPC repository integration tests")
	}

	ctx := context.Background()
	pool, err := repository.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	_, currentFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../"))
	migrationsDir := filepath.Join(repoRoot, "db", "migrations")
	if err := indismigrate.Migrate(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}

	zkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":  true,
			"reason": "",
		})
	}))

	repo := repository.New(pool)
	svc := service.New(repo, zkServer.URL)
	h := New(svc)

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	electoralv1.RegisterElectoralServiceServer(grpcServer, h)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		zkServer.Close()
		pool.Close()
		t.Fatalf("dial grpc: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
		zkServer.Close()
		pool.Close()
	}

	client := electoralv1.NewElectoralServiceClient(conn)
	return ctx, client, pool, cleanup
}

func TestSubmitRemoteBallotGRPCWithRealRepository(t *testing.T) {
	t.Parallel()

	ctx, client, pool, cleanup := setupRemoteBallotIntegration(t)
	defer cleanup()

	registerResp, err := client.RegisterElection(ctx, &electoralv1.RegisterElectionRequest{
		Name:     "Remote Voting Integration Election",
		OpensAt:  time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		ClosesAt: time.Now().UTC().Add(25 * time.Hour).Format(time.RFC3339),
		AdminDid: "did:indis:admin:integration",
	})
	if err != nil {
		t.Fatalf("register election: %v", err)
	}

	firstReq := &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        registerResp.GetElectionId(),
		NullifierHash:     "integration-nullifier-1",
		EncryptedVote:     []byte("encrypted-vote-1"),
		ZkProof:           []byte("zk-proof-1"),
		ClientAttestation: []byte("device-attestation"),
		SubmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("integration-nonce-1"),
	}

	remoteResp, err := client.SubmitRemoteBallot(ctx, firstReq)
	if err != nil {
		t.Fatalf("submit remote ballot: %v", err)
	}
	if remoteResp.GetReceiptHash() == "" || remoteResp.GetAcceptedAt() == "" {
		t.Fatalf("expected receipt and accepted_at in response")
	}

	var network string
	var attestationHash, nonceHash string
	if err := pool.QueryRow(ctx,
		`SELECT remote_network, client_attestation_hash, transport_nonce_hash
		 FROM ballots
		 WHERE election_id = $1 AND nullifier_hash = $2`,
		registerResp.GetElectionId(),
		"integration-nullifier-1",
	).Scan(&network, &attestationHash, &nonceHash); err != nil {
		t.Fatalf("query persisted remote metadata: %v", err)
	}
	if network != "mobile" || attestationHash == "" || nonceHash == "" {
		t.Fatalf("unexpected persisted remote metadata: network=%q attestation=%q nonce=%q", network, attestationHash, nonceHash)
	}

	secondReq := &electoralv1.SubmitRemoteBallotRequest{
		ElectionId:        registerResp.GetElectionId(),
		NullifierHash:     "integration-nullifier-2",
		EncryptedVote:     []byte("encrypted-vote-2"),
		ZkProof:           []byte("zk-proof-2"),
		ClientAttestation: []byte("device-attestation"),
		SubmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Network:           "mobile",
		TransportNonce:    []byte("integration-nonce-1"),
	}

	_, err = client.SubmitRemoteBallot(ctx, secondReq)
	if err == nil {
		t.Fatalf("expected replay nonce rejection")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal code for replay rejection, got %s", st.Code())
	}
}

func TestSubmitRemoteBallotConcurrentLoadAndReplayPressure(t *testing.T) {
	t.Parallel()

	ctx, client, pool, cleanup := setupRemoteBallotIntegration(t)
	defer cleanup()

	registerResp, err := client.RegisterElection(ctx, &electoralv1.RegisterElectionRequest{
		Name:     "Remote Voting Load Test Election",
		OpensAt:  time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		ClosesAt: time.Now().UTC().Add(25 * time.Hour).Format(time.RFC3339),
		AdminDid: "did:indis:admin:load",
	})
	if err != nil {
		t.Fatalf("register election: %v", err)
	}

	const workers = 24
	var successCount atomic.Int32
	var replayRejectCount atomic.Int32
	var otherFailureCount atomic.Int32

	baseTime := time.Now().UTC().Format(time.RFC3339)
	replayedNonce := []byte("shared-replay-nonce")

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()

			nonce := []byte(fmt.Sprintf("nonce-%02d", i))
			if i%6 == 0 {
				nonce = replayedNonce
			}

			_, callErr := client.SubmitRemoteBallot(ctx, &electoralv1.SubmitRemoteBallotRequest{
				ElectionId:        registerResp.GetElectionId(),
				NullifierHash:     fmt.Sprintf("load-nullifier-%02d", i),
				EncryptedVote:     []byte(fmt.Sprintf("encrypted-vote-%02d", i)),
				ZkProof:           []byte("zk-proof-load"),
				ClientAttestation: []byte("device-attestation-load"),
				SubmittedAt:       baseTime,
				Network:           "mobile",
				TransportNonce:    nonce,
			})

			if callErr == nil {
				successCount.Add(1)
				return
			}

			st, ok := status.FromError(callErr)
			if !ok {
				otherFailureCount.Add(1)
				return
			}
			if st.Code() == codes.Internal {
				replayRejectCount.Add(1)
				return
			}
			otherFailureCount.Add(1)
		}()
	}
	wg.Wait()

	if otherFailureCount.Load() != 0 {
		t.Fatalf("unexpected failures during concurrent submit: %d", otherFailureCount.Load())
	}
	if successCount.Load() == 0 {
		t.Fatalf("expected at least one successful remote submission")
	}
	if replayRejectCount.Load() == 0 {
		t.Fatalf("expected replay-pressure rejections from shared nonce")
	}

	var ballotCount int64
	if err := pool.QueryRow(ctx, `SELECT ballot_count FROM elections WHERE id = $1`, registerResp.GetElectionId()).Scan(&ballotCount); err != nil {
		t.Fatalf("query ballot_count: %v", err)
	}
	if ballotCount != int64(successCount.Load()) {
		t.Fatalf("ballot_count mismatch: expected=%d got=%d", successCount.Load(), ballotCount)
	}
}
