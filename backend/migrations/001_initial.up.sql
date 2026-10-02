CREATE TABLE competitions (id uuid PRIMARY KEY, name text NOT NULL);
CREATE TABLE teams (id uuid PRIMARY KEY, name text NOT NULL);
CREATE TABLE matches (
 id uuid PRIMARY KEY, competition_id uuid NOT NULL REFERENCES competitions(id),
 home_id uuid NOT NULL REFERENCES teams(id), away_id uuid NOT NULL REFERENCES teams(id),
 kickoff timestamptz NOT NULL, provider_name text NOT NULL, external_id text NOT NULL,
 data jsonb NOT NULL, UNIQUE(provider_name, external_id)
);
CREATE INDEX matches_kickoff ON matches(kickoff);
CREATE TABLE odds_snapshots (
 id uuid PRIMARY KEY, match_id uuid NOT NULL REFERENCES matches(id),
 captured_at timestamptz NOT NULL, source_key text NOT NULL UNIQUE, data jsonb NOT NULL
);
CREATE INDEX odds_match_time ON odds_snapshots(match_id,captured_at DESC);
CREATE TABLE lineup_snapshots (
 id uuid PRIMARY KEY, match_id uuid NOT NULL REFERENCES matches(id),
 captured_at timestamptz NOT NULL, source_key text NOT NULL UNIQUE, data jsonb NOT NULL
);
CREATE INDEX lineup_match_time ON lineup_snapshots(match_id,captured_at DESC);
CREATE TABLE predictions (
 id uuid PRIMARY KEY, match_id uuid NOT NULL REFERENCES matches(id),
 odds_snapshot_id uuid NOT NULL REFERENCES odds_snapshots(id),
 model_version text NOT NULL, generated_at timestamptz NOT NULL, data jsonb NOT NULL
);
CREATE INDEX predictions_match_time ON predictions(match_id,generated_at DESC);
CREATE TABLE recommendations (
 id uuid PRIMARY KEY, match_id uuid NOT NULL REFERENCES matches(id),
 odds_snapshot_id uuid REFERENCES odds_snapshots(id), prediction_id uuid REFERENCES predictions(id),
 model_version text NOT NULL, generated_at timestamptz NOT NULL, data jsonb NOT NULL
);
CREATE INDEX recommendations_match_time ON recommendations(match_id,generated_at DESC);
CREATE TABLE user_picks (
 id uuid PRIMARY KEY, recommendation_id uuid NOT NULL REFERENCES recommendations(id),
 match_id uuid NOT NULL REFERENCES matches(id), market text NOT NULL, selection text NOT NULL,
 line numeric, picked_at timestamptz NOT NULL, cancelled_at timestamptz,
 settled_at timestamptz, result text, net_profit_units double precision,
 data jsonb NOT NULL,
 CHECK (result IS NULL OR result IN ('win','half-win','push','half-loss','loss','void')),
 CHECK ((settled_at IS NULL AND result IS NULL AND net_profit_units IS NULL) OR
        (settled_at IS NOT NULL AND result IS NOT NULL AND net_profit_units IS NOT NULL))
);
CREATE UNIQUE INDEX user_picks_active ON user_picks(match_id,market,selection,COALESCE(line,999999)) WHERE cancelled_at IS NULL;
CREATE INDEX picks_time ON user_picks(picked_at DESC);
CREATE TABLE match_results (
 match_id uuid PRIMARY KEY REFERENCES matches(id), recorded_at timestamptz NOT NULL, data jsonb NOT NULL
);
