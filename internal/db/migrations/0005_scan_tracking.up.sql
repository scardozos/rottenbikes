-- Track whether submissions originated from an actual QR code scan.
-- Moderation signal: bikes/reviews submitted without a scan are worth a closer look.
ALTER TABLE bikes ADD COLUMN was_scanned BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE reviews ADD COLUMN was_scanned BOOLEAN NOT NULL DEFAULT FALSE;

-- Server-side record of authenticated QR scans: one row per poster + bike.
-- The review's was_scanned flag is derived from this table at insert time,
-- so clients cannot spoof it. Rows are refreshed on every scan (upsert).
CREATE TABLE scan_events (
    poster_id         BIGINT      NOT NULL REFERENCES posters(poster_id) ON DELETE CASCADE,
    bike_numerical_id TEXT        NOT NULL REFERENCES bikes(numerical_id) ON DELETE CASCADE,
    scanned_ts        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (poster_id, bike_numerical_id)
);

CREATE INDEX idx_scan_events_bike ON scan_events (bike_numerical_id);
