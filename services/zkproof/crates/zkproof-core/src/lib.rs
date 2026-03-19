//! INDIS ZK Proof Core Library
//!
//! Core zero-knowledge proof generation and verification logic.
//!
//! # Proof Systems
//!
//! | Use Case | System | Rationale |
//! |----------|--------|-----------|
//! | Standard credential verification | Groth16 (ZK-SNARK) | Fast proof generation (<3s on mid-range phone) |
//! | Electoral / referendum verification | ZK-STARK | Post-quantum security; no trusted setup |
//! | Batch credential operations | PLONK | Universal trusted setup; efficient for bulk ops |
//! | Anonymous testimony (Justice) | Bulletproofs | No trusted setup; range proofs |
//!
//! # Performance Targets (PRD §FR-003)
//!
//! - Standard proof generation: 2s target, 5s max
//! - Electoral STARK generation: 5s target, 15s max
//! - Proof verification: 200ms target, 500ms max

use serde::{Deserialize, Serialize};
use thiserror::Error;

pub mod groth16;
pub mod stark;

pub use groth16::DevelopmentGroth16Engine;
pub use stark::DevelopmentStarkEngine;

/// Errors that can occur during ZK proof operations.
#[derive(Error, Debug)]
pub enum ZkError {
    #[error("proof generation failed: {0}")]
    GenerationFailed(String),

    #[error("proof verification failed: {0}")]
    VerificationFailed(String),

    #[error("invalid circuit: {0}")]
    InvalidCircuit(String),

    #[error("unsupported proof system: {0}")]
    UnsupportedSystem(String),
}

/// Supported ZK proof systems.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProofSystem {
    /// Groth16 ZK-SNARK — standard credential verification
    Groth16,
    /// ZK-STARK — electoral/referendum (post-quantum)
    Stark,
    /// PLONK — batch credential operations
    Plonk,
    /// Bulletproofs — anonymous testimony
    Bulletproofs,
}

/// A generated zero-knowledge proof.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Proof {
    /// The proof system used.
    pub system: ProofSystem,
    /// Serialized proof data.
    pub data: Vec<u8>,
    /// Public inputs to the proof.
    pub public_inputs: Vec<Vec<u8>>,
}

/// Result of verifying a ZK proof.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    /// Whether the proof is valid.
    pub valid: bool,
    /// The proof system used.
    pub system: ProofSystem,
}

/// Trait for ZK proof generation engines.
pub trait ProofGenerator {
    /// Generate a ZK proof for the given circuit and inputs.
    fn generate(
        &self,
        circuit_id: &str,
        private_inputs: &[Vec<u8>],
        public_inputs: &[Vec<u8>],
    ) -> Result<Proof, ZkError>;
}

/// Trait for ZK proof verification engines.
pub trait ProofVerifier {
    /// Verify a ZK proof against the verification key and public inputs.
    fn verify(
        &self,
        proof: &Proof,
        verification_key: &[u8],
        public_inputs: &[Vec<u8>],
    ) -> Result<VerificationResult, ZkError>;
}
