// Command pqc-migrate re-signs INDIS Verifiable Credentials with
// CRYSTALS-Dilithium3 post-quantum signatures.
//
// Usage:
//
//	pqc-migrate --database-url <DSN> --issuer-did <DID> --dry-run
//
// Ref: INDIS PRD §4.3 — Post-Quantum Cryptography migration path.
// Build with: go build -tags circl -o pqc-migrate ./tools/pqc-migrate/
// (requires filippo.io/circl in go.sum)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mohamadsolouki/INDIS/pkg/crypto"
	"github.com/mohamadsolouki/INDIS/pkg/vc"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		databaseURL = flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string for credential DB")
		issuerDID   = flag.String("issuer-did", os.Getenv("ISSUER_DID"), "Issuer DID used when re-signing credentials")
		batchSize   = flag.Int("batch-size", 100, "Number of credentials to re-sign per DB batch")
		dryRun      = flag.Bool("dry-run", false, "Validate and count without writing updated credentials")
	)
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("--database-url or DATABASE_URL env var is required")
	}
	if *issuerDID == "" {
		log.Fatal("--issuer-did or ISSUER_DID env var is required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	defer pool.Close()

	// Generate (or load) Dilithium3 key pair.
	// In production wire pkg/hsm.VaultKeyManager instead.
	dilithiumKP, err := crypto.GenerateDilithiumKeyPair()
	if err != nil {
		log.Fatalf("dilithium key generation: %v", err)
	}
	_ = dilithiumKP.PublicKey // stored in DID document; not persisted here
	privKey := dilithiumKP.PrivateKey

	log.Printf("pqc-migrate: starting (dry-run=%v, batch=%d, issuer=%s)", *dryRun, *batchSize, *issuerDID)

	offset := 0
	total := 0
	updated := 0
	errs := 0

	for {
		rows, err := pool.Query(ctx,
			`SELECT id, data FROM credentials WHERE status = 'active' ORDER BY created_at LIMIT $1 OFFSET $2`,
			*batchSize, offset,
		)
		if err != nil {
			log.Fatalf("query batch at offset %d: %v", offset, err)
		}

		type row struct {
			id   string
			data []byte
		}
		var batch []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.data); err != nil {
				log.Printf("scan row: %v", err)
				errs++
				continue
			}
			batch = append(batch, r)
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		for _, r := range batch {
			total++
			var credential vc.VerifiableCredential
			if err := json.Unmarshal(r.data, &credential); err != nil {
				log.Printf("unmarshal credential %s: %v", r.id, err)
				errs++
				continue
			}

			// Re-sign with Dilithium3.
			verificationMethod := *issuerDID + "#dilithium3-key-1"
			updated_cred, err := vc.IssueWithSigner(
				vc.CredentialType(credential.Type[len(credential.Type)-1]),
				*issuerDID,
				verificationMethod,
				credential.CredentialSubject,
				credential.ValidFrom,
				credential.ValidUntil,
				func(data []byte) ([]byte, error) {
					return crypto.SignDilithium(privKey, data)
				},
			)
			if err != nil {
				log.Printf("re-sign credential %s: %v", r.id, err)
				errs++
				continue
			}

			newData, err := json.Marshal(updated_cred)
			if err != nil {
				log.Printf("marshal re-signed credential %s: %v", r.id, err)
				errs++
				continue
			}

			if *dryRun {
				log.Printf("[dry-run] would update credential %s (proof_system=dilithium3)", r.id)
				updated++
				continue
			}

			_, err = pool.Exec(ctx,
				`UPDATE credentials SET data = $1, updated_at = $2 WHERE id = $3`,
				newData, time.Now().UTC(), r.id,
			)
			if err != nil {
				log.Printf("update credential %s: %v", r.id, err)
				errs++
				continue
			}
			updated++
		}

		offset += len(batch)
		log.Printf("pqc-migrate: processed %d / batch offset %d (updated=%d, errors=%d)", total, offset, updated, errs)

		if len(batch) < *batchSize {
			break
		}
	}

	fmt.Printf("\npqc-migrate complete: total=%d updated=%d errors=%d dry-run=%v\n", total, updated, errs, *dryRun)
	if errs > 0 {
		os.Exit(1)
	}
}
