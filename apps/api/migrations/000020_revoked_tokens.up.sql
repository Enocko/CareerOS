-- Server-side revocation for logged-out session tokens (JWT jti).
CREATE TABLE revoked_tokens (
    token_id   TEXT        PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);
