import { afterEach, expect, it, vi } from "vitest";
import { requestAll } from "./api";
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it("collects all pages of independent History lists", async () => {
  vi.stubEnv("NEXT_PUBLIC_API_URL", "http://backend.test");
  const response = (items: number[], total: number, offset: number) => ({
    items,
    total,
    offset,
    limit: 2,
  });
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        matches: response([1], 1, 0),
        recommendations: response([10, 11], 3, 0),
        picks: response([20, 21], 4, 0),
        stats: { roi: 0.12 },
      }),
    })
    .mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        matches: response([], 1, 2),
        recommendations: response([12], 3, 2),
        picks: response([22, 23], 4, 2),
        stats: { roi: 0.12 },
      }),
    });
  vi.stubGlobal("fetch", fetchMock);
  const data = await requestAll<{
    recommendations: { items: number[] };
    picks: { items: number[] };
    matches: { items: number[] };
    stats: { roi: number };
  }>("/api/v1/history?limit=2&date=2026-10-02");
  expect(data.recommendations.items).toEqual([10, 11, 12]);
  expect(data.picks.items).toEqual([20, 21, 22, 23]);
  expect(data.matches.items).toEqual([1]);
  expect(data.stats.roi).toBe(0.12);
  expect(fetchMock.mock.calls[1][0]).toContain("offset=2");
  expect(fetchMock.mock.calls[1][0]).toContain("date=2026-10-02");
});
