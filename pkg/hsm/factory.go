// Package hsm — factory.go
// Environment-driven KeyManager factory.
package hsm

import "os"

// New creates a KeyManager driven by environment variables.
//
// Environment variables:
//
//	HSM_BACKEND         "software" (default) or "vault"
//	VAULT_ADDR          Vault server URL (required when HSM_BACKEND=vault)
//	VAULT_TOKEN         Vault token (required when HSM_BACKEND=vault)
//	VAULT_TRANSIT_MOUNT Transit engine mount path (default: "transit")
//
// When HSM_BACKEND is absent or set to "software", a new SoftwareKeyManager
// is returned. This is suitable for development and CI but must never be used
// in production because keys are held in process memory and are lost on
// restart.
//
// When HSM_BACKEND=vault, a VaultKeyManager is constructed from the above
// environment variables. The caller is responsible for ensuring that
// VAULT_ADDR and VAULT_TOKEN are set; if they are empty the Vault client will
// fail on the first operation.
func New() KeyManager {
	backend := os.Getenv("HSM_BACKEND")
	if backend == "" || backend == "software" {
		return NewSoftwareKeyManager()
	}
	addr := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")
	mount := os.Getenv("VAULT_TRANSIT_MOUNT")
	return NewVaultKeyManager(addr, token, mount)
}
