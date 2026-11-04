import type { LogLine } from "../types";

let seq = 0;

/** Docker --timestamps lines look like: 2024-01-02T03:04:05.123456789Z message */
export function parseLogLine(raw: string): LogLine {
  const match = raw.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z)\s?(.*)$/);
  seq += 1;
  if (!match) {
    return { id: `${seq}`, raw, body: raw };
  }
  return { id: `${seq}`, raw, ts: match[1], body: match[2] ?? "" };
}

export function resetLogSeq() {
  seq = 0;
}
