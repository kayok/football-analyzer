export interface Selection {
  market: "1X2" | "AH" | "OU";
  selection: string;
  line: number | null;
}
export interface Odds extends Selection {
  id: string;
  match_id: string;
  odds: number;
  bookmaker: string;
  captured_at: string;
  phase: string;
}
export interface Distribution {
  p_win: number;
  p_half_win: number;
  p_push: number;
  p_half_loss: number;
  p_loss: number;
}
export interface Prediction extends Selection {
  id: string;
  match_id: string;
  odds_snapshot_id: string;
  distribution: Distribution;
  probability: number;
  ev: number;
  fair_odds: number | null;
  minimum_acceptable_odds: number | null;
  market_implied_probability: number;
  margin_removed_probability: number | null;
  one_x_two: number[];
  expected_goals_home: number;
  expected_goals_away: number;
  model_version: string;
  generated_at: string;
}
export interface Recommendation {
  id: string;
  match_id: string;
  status: "PLAY" | "WATCH" | "PASS";
  reasons: string[];
  reason_code: string;
  generated_at: string;
  model_version: string;
  odds: Odds | null;
  prediction: Prediction | null;
}
export interface Match {
  provider_name?: string;
  id: string;
  home: string;
  away: string;
  competition: string;
  kickoff: string;
  status: string;
  expected_goals_home: number;
  expected_goals_away: number;
}
export interface Result {
  match_id: string;
  home: number;
  away: number;
  status: string;
}
export interface Pick extends Selection {
  id: string;
  match_id: string;
  recommendation_id: string;
  odds_at_pick: number;
  probability_at_pick: number;
  ev_at_pick: number;
  bookmaker: string;
  model_version: string;
  stake_units: number;
  picked_at: string;
  cancelled_at: string | null;
  settled_at: string | null;
  result: string | null;
  net_profit_units: number | null;
  demo: boolean;
  match: Match;
}
export interface Card extends Match {
  recommendation: Recommendation | null;
  pick: Pick | null;
  result: Result | null;
}
export interface Lineup {
  id: string;
  captured_at: string;
  phase: string;
  home: Team;
  away: Team;
}
export interface Team {
  starting_xi: string[];
  substitutes: string[];
  injuries: string[];
  suspensions: string[];
}
export interface Detail extends Card {
  odds_snapshots: Odds[];
  prediction_snapshots: Prediction[];
  recommendation_snapshots: Recommendation[];
  lineup_snapshots: Lineup[];
}
export interface Page<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
}
export interface History {
  matches: Page<Card>;
  recommendations: Page<Recommendation>;
  picks: Page<Pick>;
  results: Page<Result>;
  stats: {
    profit_units: number;
    settled_stake_units: number;
    roi: number | null;
    settled_picks: number;
    brier_by_model: Record<string, number>;
  };
}
