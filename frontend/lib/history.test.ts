import { expect, it } from "vitest";
import { groupPicksByDate, groupRecommendationsByDate } from "./history";

it("groups local pick dates and sorts newest first without changing the source", () => {
  const picks = [
    { id: "before", picked_at: "2026-10-05T16:59:00Z" },
    { id: "midnight", picked_at: "2026-10-05T17:00:00Z" },
    { id: "latest", picked_at: "2026-10-05T17:30:00Z" },
  ];
  const groups = groupPicksByDate(picks, "Asia/Bangkok");
  expect(groups.map((group) => group.date)).toEqual([
    "2026-10-06",
    "2026-10-05",
  ]);
  expect(groups[0].items.map((pick) => pick.id)).toEqual([
    "latest",
    "midnight",
  ]);
  expect(groups[1].items[0].id).toBe("before");
  expect(picks.map((pick) => pick.id)).toEqual([
    "before",
    "midnight",
    "latest",
  ]);
  expect(
    groupPicksByDate(picks, "America/New_York").map((group) => group.date),
  ).toEqual(["2026-10-05"]);
});

it("keeps both occurrences of a repeated DST hour in the same local date", () => {
  const groups = groupPicksByDate(
    [
      { picked_at: "2026-11-01T05:30:00Z" },
      { picked_at: "2026-11-01T06:30:00Z" },
    ],
    "America/New_York",
  );
  expect(groups).toHaveLength(1);
  expect(groups[0].date).toBe("2026-11-01");
  expect(groups[0].items).toHaveLength(2);
  expect(groupPicksByDate([], "Asia/Bangkok")).toEqual([]);
});

it("groups recommendations by their generation date rather than kickoff", () => {
  const recommendations = [
    {
      id: "earlier",
      generated_at: "2026-10-05T16:59:00Z",
      kickoff: "2026-10-06T12:00:00Z",
    },
    {
      id: "later",
      generated_at: "2026-10-05T17:00:00Z",
      kickoff: "2026-10-06T12:00:00Z",
    },
  ];
  const groups = groupRecommendationsByDate(recommendations, "Asia/Bangkok");
  expect(groups.map((group) => group.date)).toEqual([
    "2026-10-06",
    "2026-10-05",
  ]);
  expect(groups[0].items[0].id).toBe("later");
});
