-- TelHawk Query Service - Saved Searches (immutable, versioned)

CREATE TABLE IF NOT EXISTS saved_searches (
    id UUID NOT NULL,                  -- Stable identifier (UUID v7)
    version_id UUID PRIMARY KEY,       -- Version-specific UUID (UUID v7)
    owner_id UUID,                     -- Reference to auth.users(id); NULL for global
    created_by UUID NOT NULL,          -- Who created this version (auth.users(id))
    name TEXT NOT NULL,                -- Display name (not unique)
    query JSONB NOT NULL,              -- OpenSearch Query DSL JSON
    filters JSONB,                     -- Optional: time windows, tags, ui state
    is_global BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Lifecycle timestamps (immutable pattern)
    disabled_at TIMESTAMP,
    disabled_by UUID,
    hidden_at TIMESTAMP,               -- Hidden from UI (soft delete feel)
    hidden_by UUID
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_saved_searches_id_created ON saved_searches(id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_saved_searches_id ON saved_searches(id);
CREATE INDEX IF NOT EXISTS idx_saved_searches_version_id ON saved_searches(version_id);
CREATE INDEX IF NOT EXISTS idx_saved_searches_owner ON saved_searches(owner_id);
CREATE INDEX IF NOT EXISTS idx_saved_searches_created ON saved_searches(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_saved_searches_active ON saved_searches(id, created_at DESC)
    WHERE hidden_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_saved_searches_query_gin ON saved_searches USING GIN (query);

COMMENT ON TABLE saved_searches IS 'Versioned saved searches with immutable lifecycle (id + version_id)';
COMMENT ON COLUMN saved_searches.id IS 'Stable identifier grouping all versions';
COMMENT ON COLUMN saved_searches.version_id IS 'Version-specific UUID (UUID v7, time-ordered)';
COMMENT ON COLUMN saved_searches.created_by IS 'User who authored this version';
COMMENT ON COLUMN saved_searches.hidden_at IS 'When search was hidden (NULL = visible)';
