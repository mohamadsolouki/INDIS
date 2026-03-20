-- INDIS Enrollment Service — Social attestation minimum co-attestor constraint
-- Migration: 011
-- Applies to: PostgreSQL 16+
--
-- PRD §FR-005.3 (Social Attestation Pathway):
--   "At least 3 community co-attestors must have approved before enrollment can complete."
--
-- The service layer already enforces this in Go, but a service bypass (direct DB write,
-- admin tooling, future migration) could produce under-attested social enrollments.
-- This trigger adds a hard database-level guarantee.

CREATE OR REPLACE FUNCTION enforce_social_attestation_minimum()
RETURNS TRIGGER AS $$
BEGIN
    -- Only fire when transitioning to 'completed' on a social pathway enrollment.
    IF NEW.status = 'completed'
       AND NEW.pathway = 'social'
       AND NEW.attestor_count < 3
    THEN
        RAISE EXCEPTION
            'Social pathway enrollment % cannot complete with only % attestor(s); minimum is 3 (PRD §FR-005.3)',
            NEW.id, NEW.attestor_count;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop if re-running migration idempotently.
DROP TRIGGER IF EXISTS trg_social_attestation_minimum ON enrollments;

CREATE TRIGGER trg_social_attestation_minimum
    BEFORE UPDATE OF status ON enrollments
    FOR EACH ROW
    EXECUTE FUNCTION enforce_social_attestation_minimum();

COMMENT ON FUNCTION enforce_social_attestation_minimum() IS
    'Enforces PRD §FR-005.3: social pathway enrollments require ≥3 co-attestors before completing.';
