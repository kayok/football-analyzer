import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { RecommendationCard } from "./recommendation-card";
import type { Card } from "../lib/types";
vi.mock("../lib/api", () => ({
  selectPick: vi.fn().mockResolvedValue({ id: "pick" }),
}));
import { selectPick } from "../lib/api";
const card: Card = {
  id: "match",
  home: "Liverpool",
  away: "Arsenal",
  competition: "Premier League",
  kickoff: "2026-10-02T12:00:00Z",
  status: "scheduled",
  expected_goals_home: 1.8,
  expected_goals_away: 1.1,
  pick: null,
  result: null,
  recommendation: {
    id: "rec",
    match_id: "match",
    status: "PLAY",
    reason_code: "VALUE",
    reasons: ["ราคามีความคุ้มค่า"],
    generated_at: "2026-10-02T10:00:00Z",
    model_version: "v1",
    odds: {
      id: "odds",
      match_id: "match",
      market: "1X2",
      selection: "home",
      line: null,
      bookmaker: "MockBook",
      odds: 2,
      captured_at: "2026-10-02T10:00:00Z",
      phase: "manual",
    },
    prediction: {
      id: "p",
      match_id: "match",
      odds_snapshot_id: "odds",
      market: "1X2",
      selection: "home",
      line: null,
      distribution: {
        p_win: 0.56,
        p_half_win: 0,
        p_push: 0,
        p_half_loss: 0,
        p_loss: 0.44,
      },
      probability: 0.56,
      ev: 0.12,
      fair_odds: 1.78,
      minimum_acceptable_odds: 1.88,
      market_implied_probability: 0.5,
      margin_removed_probability: 0.48,
      one_x_two: [0.56, 0.24, 0.2],
      expected_goals_home: 1.8,
      expected_goals_away: 1.1,
      model_version: "v1",
      generated_at: "2026-10-02T10:00:00Z",
    },
  },
};
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});
describe("recommendation card", () => {
  it("shows backend values and stores a pick only after clicking", async () => {
    const reload = vi.fn();
    render(<RecommendationCard card={card} onPicked={reload} />);
    expect(screen.getByText("+12.0%")).toBeInTheDocument();
    expect(selectPick).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "เลือกเล่น" }));
    await waitFor(() => expect(selectPick).toHaveBeenCalledWith("rec"));
    expect(reload).toHaveBeenCalled();
  });
  it("PASS cannot be selected", () => {
    render(
      <RecommendationCard
        card={{
          ...card,
          recommendation: { ...card.recommendation!, status: "PASS" },
        }}
        onPicked={vi.fn()}
      />,
    );
    expect(screen.getByText("PASS")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "เลือกเล่น" })).toBeNull();
  });
  it("not playing dismisses without writing a pick", () => {
    render(<RecommendationCard card={card} onPicked={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "ไม่เล่น" }));
    expect(selectPick).not.toHaveBeenCalled();
    expect(
      screen.getByRole("button", { name: "แสดงอีกครั้ง" }),
    ).toBeInTheDocument();
  });
  it("shows a failed save and allows retry", async () => {
    vi.mocked(selectPick).mockRejectedValueOnce(new Error("คำแนะนำหมดอายุ"));
    render(<RecommendationCard card={card} onPicked={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "เลือกเล่น" }));
    await screen.findByRole("alert");
    expect(screen.getByText("คำแนะนำหมดอายุ")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "เลือกเล่น" })).toBeEnabled();
  });
});
