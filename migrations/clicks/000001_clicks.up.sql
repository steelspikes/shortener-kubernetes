-- Base de datos de clics. Sin FK a links: viven en otra base (code es solo una referencia lógica).
CREATE TABLE clicks (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(16) NOT NULL,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_hash    TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_clicks_code_clicked_at ON clicks (code, clicked_at);
