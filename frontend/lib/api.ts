import type { Pick } from "./types";
export async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const base = process.env.NEXT_PUBLIC_API_URL;
  if (!base)
    throw new Error(
      "ยังไม่ได้ตั้งค่า NEXT_PUBLIC_API_URL กรุณาเปิดแอปผ่าน make frontend",
    );
  const response = await fetch(`${base}${path}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options?.headers },
    cache: "no-store",
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as {
      error?: { message?: string };
    } | null;
    throw new Error(
      body?.error?.message || "โหลดข้อมูลไม่ได้ กรุณาตรวจว่า backend เปิดอยู่",
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
export function selectPick(id: string) {
  return request<Pick>("/api/v1/user-picks", {
    method: "POST",
    body: JSON.stringify({ recommendation_id: id }),
  });
}
export function cancelPick(id: string) {
  return request<void>(`/api/v1/user-picks/${id}`, { method: "DELETE" });
}
export function timezone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Bangkok";
}

type Collection = {
  items: unknown[];
  total: number;
  offset: number;
  limit: number;
};
function isCollection(value: unknown): value is Collection {
  return (
    typeof value === "object" &&
    value !== null &&
    "items" in value &&
    Array.isArray(value.items) &&
    "total" in value &&
    typeof value.total === "number" &&
    "limit" in value &&
    typeof value.limit === "number"
  );
}
// Fetch all pages for a list or a response containing multiple paginated collections.
export async function requestAll<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const first = await request<unknown>(path, options);
  const single = isCollection(first);
  const collections: Record<string, Collection> = {};
  if (single) collections.items = first;
  else if (typeof first === "object" && first !== null) {
    for (const [key, value] of Object.entries(first))
      if (isCollection(value)) collections[key] = value;
  }
  const values = Object.values(collections);
  if (!values.length) return first as T;
  const step = values[0].limit;
  const total = Math.max(...values.map((p) => p.total));
  const [route, query] = path.split("?");
  const params = new URLSearchParams(query);
  for (let offset = step; offset < total; offset += step) {
    params.set("offset", String(offset));
    const next = await request<unknown>(`${route}?${params}`, options);
    if (single && isCollection(next))
      collections.items.items.push(...next.items);
    else if (typeof next === "object" && next !== null) {
      for (const [key, value] of Object.entries(next))
        if (collections[key] && isCollection(value))
          collections[key].items.push(...value.items);
    }
  }
  return first as T;
}
