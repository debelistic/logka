import type { Container } from "./types";

export async function fetchHealth(): Promise<{ ok: boolean; docker?: string }> {
  const res = await fetch("/api/health");
  return res.json();
}

export async function fetchContainers(): Promise<Container[]> {
  const res = await fetch("/api/containers");
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `Failed to list containers (${res.status})`);
  }
  const data = await res.json();
  return data.containers ?? [];
}

export function openLogStream(
  id: string,
  opts: { tail?: string; follow?: boolean } = {},
): EventSource {
  const params = new URLSearchParams();
  if (opts.tail) params.set("tail", opts.tail);
  if (opts.follow === false) params.set("follow", "false");
  const qs = params.toString();
  return new EventSource(`/api/containers/${encodeURIComponent(id)}/logs${qs ? `?${qs}` : ""}`);
}
