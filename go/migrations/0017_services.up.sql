-- theblock — P3 Servisler: önbellek + 81 il kataloğu (0017)
CREATE TABLE services_cache (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind       TEXT NOT NULL,
    key        TEXT NOT NULL,
    data       JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kind, key)
);

CREATE TABLE cities (
    plaka INT PRIMARY KEY,
    name  TEXT NOT NULL,
    slug  TEXT NOT NULL UNIQUE,
    lat   DOUBLE PRECISION NOT NULL,
    lon   DOUBLE PRECISION NOT NULL,
    sort  INT NOT NULL DEFAULT 0
);
