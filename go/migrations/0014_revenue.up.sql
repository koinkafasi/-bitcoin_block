-- theblock — P5-D Gelir analitiği (0014)
CREATE TABLE content_revenue (
    content_id BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    revenue_est NUMERIC(12,4) NOT NULL DEFAULT 0,
    impressions BIGINT NOT NULL DEFAULT 0,
    rpm_est     NUMERIC(8,2)  NOT NULL DEFAULT 0,
    cost_est    NUMERIC(12,4) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (section, key, value) VALUES
    ('gelir','varsayilan_rpm','"3.0"'::jsonb),
    ('gelir','demo_modu','true'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
