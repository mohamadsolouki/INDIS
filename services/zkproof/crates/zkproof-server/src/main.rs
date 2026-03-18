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
use zkproof_circuits::VoterEligibilityStarkAir;
use zkproof_core::{DevelopmentStarkEngine, Proof, ProofSystem};
use zkproof_core::{ProofGenerator as _, ProofVerifier as _};

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

fn is_stark(proof_system: &str) -> bool {
    proof_system.eq_ignore_ascii_case("stark")
}

fn generate_stark_proof(circuit_id: &str, input_b64: &str) -> Result<String, String> {
    let input = general_purpose::STANDARD
        .decode(input_b64)
        .map_err(|e| format!("failed to decode input: {}", e))?;

    if circuit_id == "voter_eligibility" {
        serde_json::from_slice::<VoterEligibilityStarkAir>(&input)
            .map_err(|e| format!("invalid voter eligibility public inputs: {}", e))?;
    }

    let engine = DevelopmentStarkEngine;
    let proof = engine
        .generate(circuit_id, &[], &[input])
        .map_err(|e| format!("failed to generate STARK proof: {}", e))?;

    Ok(general_purpose::STANDARD.encode(proof.data))
}

fn verify_stark_proof(
    proof_b64: &str,
    election_id: Option<&String>,
    public_inputs_b64: Option<&String>,
) -> Result<(bool, String), String> {
    let proof_bytes = general_purpose::STANDARD
        .decode(proof_b64)
        .map_err(|e| format!("failed to decode proof: {}", e))?;

    let public_input_b64 = public_inputs_b64
        .ok_or_else(|| "public_inputs_b64 is required for STARK verification".to_string())?;

    let public_input = general_purpose::STANDARD
        .decode(public_input_b64)
        .map_err(|e| format!("failed to decode public inputs: {}", e))?;

    if let Some(expected_election_id) = election_id {
        let claim = serde_json::from_slice::<VoterEligibilityStarkAir>(&public_input)
            .map_err(|e| format!("invalid voter eligibility public inputs: {}", e))?;
        if claim.election_id != *expected_election_id {
            return Ok((
                false,
                "public inputs election_id does not match request election_id".to_string(),
            ));
        }
    }

    let proof = Proof {
        system: ProofSystem::Stark,
        data: proof_bytes,
        public_inputs: vec![public_input.clone()],
    };

    let engine = DevelopmentStarkEngine;
    let verification_key = election_id
        .map(|id| id.as_bytes().to_vec())
        .unwrap_or_default();

    let result = engine
        .verify(&proof, &verification_key, &[public_input])
        .map_err(|e| format!("failed to verify STARK proof: {}", e))?;

    let reason = if result.valid {
        "stark proof verified".to_string()
    } else {
        "stark proof verification failed".to_string()
    };

    Ok((result.valid, reason))
}

/// POST /prove — Generate a proof.
async fn prove(Json(req): Json<ProveRequest>) -> Result<Json<ProveResponse>, (StatusCode, String)> {
    info!(
        proof_system = &req.proof_system,
        circuit_id = &req.circuit_id,
        "generating proof"
    );

    let proof_b64 = if is_stark(&req.proof_system) {
        generate_stark_proof(&req.circuit_id, &req.input_b64)
            .map_err(|e| (StatusCode::BAD_REQUEST, e))?
    } else {
        ProofGenerator::generate_proof(&req.circuit_id, &req.input_b64)
            .map_err(|e| (StatusCode::BAD_REQUEST, e))?
    };

    Ok(Json(ProveResponse { proof_b64 }))
}

/// POST /verify — Verify a proof (handles both electoral and justice styles).
async fn verify(Json(req): Json<VerifyRequest>) -> Result<Json<VerifyResponse>, (StatusCode, String)> {
    info!(
        proof_system = &req.proof_system,
        has_election_id = req.election_id.is_some(),
        "verifying proof"
    );

    let (valid, reason) = if is_stark(&req.proof_system) {
        verify_stark_proof(
            &req.proof_b64,
            req.election_id.as_ref(),
            req.public_inputs_b64.as_ref(),
        )
        .map_err(|e| (StatusCode::BAD_REQUEST, e))?
    } else {
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

        let is_valid =
            ProofGenerator::verify_proof(&circuit_id, &req.proof_b64, &input_b64)
                .map_err(|e| (StatusCode::BAD_REQUEST, e))?;

        let verification_reason = if is_valid {
            "proof verified".to_string()
        } else {
            "proof verification failed".to_string()
        };

        (is_valid, verification_reason)
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

