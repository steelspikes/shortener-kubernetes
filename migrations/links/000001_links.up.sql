CREATE TABLE links (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(16) NOT NULL UNIQUE,
    url        TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_links_expires_at ON links (expires_at) WHERE expires_at IS NOT NULL;
