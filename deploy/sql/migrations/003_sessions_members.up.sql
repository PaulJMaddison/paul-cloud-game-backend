DROP TABLE IF EXISTS session_members;
DROP TABLE IF EXISTS sessions;

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE session_members (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, user_id)
);

CREATE INDEX idx_sessions_owner_user_id ON sessions (owner_user_id);
CREATE INDEX idx_sessions_status ON sessions (status);
CREATE INDEX idx_session_members_user_id ON session_members (user_id);
