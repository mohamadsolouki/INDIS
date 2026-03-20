package service

import (
	"context"
	"encoding/json"
	"testing"

	credentialv1 "github.com/IranProsperityProject/INDIS/api/gen/go/credential/v1"
	"google.golang.org/grpc"
)

type fakeCredentialClient struct{}

func (f *fakeCredentialClient) IssueCredential(ctx context.Context, req *credentialv1.IssueCredentialRequest, _ ...grpc.CallOption) (*credentialv1.IssueCredentialResponse, error) {
	return &credentialv1.IssueCredentialResponse{
		CredentialId: "cred-" + req.GetSubjectDid(),
		TxId:         "tx-" + req.GetSubjectDid(),
		// CredentialData is intentionally empty in the fake; govportal only uses IDs/txs.
	}, nil
}

func (f *fakeCredentialClient) VerifyCredential(context.Context, *credentialv1.VerifyCredentialRequest, ...grpc.CallOption) (*credentialv1.VerifyCredentialResponse, error) {
	return &credentialv1.VerifyCredentialResponse{Valid: true, Reason: ""}, nil
}

func (f *fakeCredentialClient) RevokeCredential(context.Context, *credentialv1.RevokeCredentialRequest, ...grpc.CallOption) (*credentialv1.RevokeCredentialResponse, error) {
	return &credentialv1.RevokeCredentialResponse{TxId: "tx-revoke"}, nil
}

func (f *fakeCredentialClient) CheckRevocationStatus(context.Context, *credentialv1.CheckRevocationStatusRequest, ...grpc.CallOption) (*credentialv1.CheckRevocationStatusResponse, error) {
	return &credentialv1.CheckRevocationStatusResponse{Revoked: false, Reason: "", RevokedAt: ""}, nil
}

func TestApproveAndExecuteBulkOperation_IssueCredential_PersistsResultSummary(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	svc := New(repo)
	svc.SetCredentialClient(&fakeCredentialClient{})

	params := json.RawMessage(`{"credential_type":1,"attributes":{"k":"v"}}`)
	targets := []string{"did:indis:1111", "did:indis:2222"}

	op, err := svc.CreateBulkOperation(context.Background(), "issue_credential", "MOI", "requester-1", targets, params)
	if err != nil {
		t.Fatalf("CreateBulkOperation: %v", err)
	}

	rec, err := svc.ApproveAndExecuteBulkOperation(context.Background(), op.ID, "approver-1")
	if err != nil {
		t.Fatalf("ApproveAndExecuteBulkOperation: %v", err)
	}
	if rec.Status != "completed" {
		t.Fatalf("expected status completed, got %q", rec.Status)
	}
	if len(rec.ResultSummary) == 0 {
		t.Fatalf("expected result_summary to be persisted")
	}

	var summary map[string]any
	if err := json.Unmarshal(rec.ResultSummary, &summary); err != nil {
		t.Fatalf("unmarshal result_summary: %v", err)
	}

	resultsAny, ok := summary["results"].([]any)
	if !ok {
		t.Fatalf("expected summary.results to be an array")
	}
	if len(resultsAny) != len(targets) {
		t.Fatalf("expected %d results, got %d", len(targets), len(resultsAny))
	}
}

