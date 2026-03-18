//! INDIS ZK Proof Server
//!
//! HTTP server for zero-knowledge proof generation and verification.
//! Supports Groth16, PLONK, STARK, and Bulletproofs proof systems.
//!
//! Baseline implementation: dummy proofs using SHA3 hashing.
//! Production will use real arkworks/Winterfell/Bulletproofs.
//!
//! See INDIS PRD v1.0 §FR-003 — Zero-Knowledge Proof System.

use axum::{
    extract::Json,
    http::StatusCode,
    routing::post,
    Router,
};
use base64::{engine::general_purpose, Engine};
use serde::{Deserialize, Serialize};
use sha3::{Sha3_256, Digest};
use tracing::info;

/// HTTP request for proof generation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProveRequest {
    pub proof_system: String,
    pub circuit_id: String,
    pub input_b64: String,
}

/// HTTP response for proof generation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProveResponse {
    pub proof_b64: String,
}

/// HTTP request for proof verification (unified).
/// Handles both electoral style (with election_id and public_inputs_b64)
/// and justice style (with just proof_system and proof_b64).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyRequest {
    pub proof_system: String,
    pub proof_b64: String,
    #[serde(default)]
    pub election_id: Option<String>,
    #[serde(default)]
    pub public_inputs_b64: Option<String>,
}

/// HTTP response for proof verification.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifyResponse {
    pub valid: bool,
    pub reason: String,
}

/// Proof generator for development baseline.
/// Production will replace this with real arkworks/Winterfell implementations.
struct ProofGenerator;

impl ProofGenerator {
    /// Generate a placeholder proof by hashing the circuit ID and inputs.
    /// This is NOT cryptographically sound — it's for development only.
    fn generate_proof(circuit_id: &str, input_b64: &str) -> Result<String, String> {
        // Decode input
        let input = general_purpose::STANDARD
            .decode(input_b64)
            .map_err(|e| format!("failed to decode input: {}", e))?;

        // Hash circuit ID + input to generate a deterministic proof
        let mut hasher = Sha3_256::new();
        hasher.update(circuit_id.as_bytes());
        hasher.update(&input);
        let proof_bytes = hasher.finalize().to_vec();

        // For development, include a short "proof" that contains the hash
        // Real proofs from arkworks/Winterfell/Bulletproofs will replace this
        let mut full_proof = vec![0u8; 32 + input.len()];
        full_proof[..32].copy_from_slice(&proof_bytes);
        full_proof[32..].copy_from_slice(&input);

        Ok(general_purpose::STANDARD.encode(&full_proof))
    }

    /// Verify a proof by reconstructing it from the input.
    /// For development: if input is not provided, always accepts the proof.
    /// Production will use real arkworks/Winterfell verifiers with actual verification keys.
    fn verify_proof(circuit_id: &str, proof_b64: &str, input_b64: &str) -> Result<bool, String> {
        let proof = general_purpose::STANDARD
            .decode(proof_b64)
            .map_err(|e| format!("failed to decode proof: {}", e))?;

        // For development: if circuit_id equals input_b64, we're in "no-input" mode
        // (justice-style proof where input wasn't provided). Accept any non-empty proof.
        if circuit_id == input_b64 {
            // No real verification - development baseline only
            return Ok(!proof.is_empty());
        }

        let input = general_purpose::STANDARD
            .decode(input_b64)
            .map_err(|e| format!("failed to decode input: {}", e))?;

        // Reconstruct expected proof from circuit ID and inputs
        let mut hasher = Sha3_256::new();
        hasher.update(circuit_id.as_bytes());
        hasher.update(&input);
        let expected_hash = hasher.finalize();

        // Check if first 32 bytes of proof match the hash
        if proof.len() < 32 {
            return Ok(false);
        }

        let expected_bytes = expected_hash.as_slice();
        Ok(&proof[..32] == &expected_bytes[..32])
    }
}

/// POST /prove — Generate a proof.
async fn prove(Json(req): Json<ProveRequest>) -> Result<Json<ProveResponse>, (StatusCode, String)> {
    info!(
        proof_system = &req.proof_system,
        circuit_id = &req.circuit_id,
        "generating proof"
    );

    let proof_b64 =
        ProofGenerator::generate_proof(&req.circuit_id, &req.input_b64)
            .map_err(|e| (StatusCode::BAD_REQUEST, e))?;

    Ok(Json(ProveResponse { proof_b64 }))
}

/// POST /verify — Verify a proof (handles both electoral and justice styles).
async fn verify(Json(req): Json<VerifyRequest>) -> Result<Json<VerifyResponse>, (StatusCode, String)> {
    info!(
        proof_system = &req.proof_system,
        has_election_id = req.election_id.is_some(),
        "verifying proof"
    );

    // Determine the circuit ID and input based on what's provided
    let (circuit_id, input_b64) = if let (Some(election_id), Some(public_inputs)) =
        (&req.election_id, &req.public_inputs_b64)
    {
        // Electoral style: use election_id as circuit identifier, public_inputs as input
        (election_id.clone(), public_inputs.clone())
    } else {
        // Justice style: use proof_system as circuit identifier
        (req.proof_system.clone(), req.proof_system.clone())
    };

    let valid =
        ProofGenerator::verify_proof(&circuit_id, &req.proof_b64, &input_b64)
            .map_err(|e| (StatusCode::BAD_REQUEST, e))?;

    let reason = if valid {
        "proof verified".to_string()
    } else {
        "proof verification failed".to_string()
    };

    Ok(Json(VerifyResponse { valid, reason }))
}

/// Health check endpoint.
async fn health() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "zkproof",
        "proof_systems": ["groth16", "stark", "plonk", "bulletproofs"]
    }))
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();

    info!("Starting INDIS ZK Proof service...");

    let app = Router::new()
        .route("/prove", post(prove))
        .route("/verify", post(verify))
        .route("/health", axum::routing::get(health));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8088").await?;
    info!("ZK Proof server listening on 0.0.0.0:8088");

    axum::serve(listener, app).await?;

    info!("INDIS ZK Proof service shut down");

    Ok(())
}

