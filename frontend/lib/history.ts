function groupByDate<T>(
  items: readonly T[],
  timestamp: (item: T) => string,
  timezone: string,
) {
  const day = new Intl.DateTimeFormat("en", {
    timeZone: timezone,
    calendar: "gregory",
    numberingSystem: "latn",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  const label = new Intl.DateTimeFormat("th-TH", {
    timeZone: timezone,
    day: "numeric",
    month: "long",
    year: "numeric",
  });
  const groups = new Map<string, { date: string; label: string; items: T[] }>();
  for (const item of [...items].sort(
    (a, b) => Date.parse(timestamp(b)) - Date.parse(timestamp(a)),
  )) {
    const instant = new Date(timestamp(item));
    const parts = day.formatToParts(instant);
    const part = (type: string) =>
      parts.find((value) => value.type === type)!.value;
    const date = `${part("year")}-${part("month")}-${part("day")}`;
    let group = groups.get(date);
    if (!group) {
      group = { date, label: label.format(instant), items: [] };
      groups.set(date, group);
    }
    group.items.push(item);
  }
  return [...groups.values()];
}

export function groupPicksByDate<T extends { picked_at: string }>(
  items: readonly T[],
  timezone: string,
) {
  return groupByDate(items, (item) => item.picked_at, timezone);
}

export function groupRecommendationsByDate<T extends { generated_at: string }>(
  items: readonly T[],
  timezone: string,
) {
  return groupByDate(items, (item) => item.generated_at, timezone);
}
