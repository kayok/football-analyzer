CREATE TABLE members (
 id uuid PRIMARY KEY,
 name text NOT NULL,
 email text NOT NULL UNIQUE CHECK (email = lower(email)),
 password_hash text NOT NULL,
 created_at timestamptz NOT NULL
);
CREATE TABLE member_sessions (
 token_hash char(64) PRIMARY KEY,
 member_id uuid NOT NULL REFERENCES members(id),
 created_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX member_sessions_expiry ON member_sessions(expires_at);
ALTER TABLE user_picks ADD COLUMN user_id uuid REFERENCES members(id);
DROP INDEX user_picks_active;
CREATE UNIQUE INDEX user_picks_active ON user_picks(COALESCE(user_id,'00000000-0000-0000-0000-000000000000'::uuid),match_id,market,selection,COALESCE(line,999999)) WHERE cancelled_at IS NULL;
CREATE INDEX user_picks_owner ON user_picks(user_id,picked_at DESC);
