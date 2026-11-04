import { useEffect, useMemo, useRef, useState } from "react";
import { fetchContainers, fetchHealth, openLogStream } from "./api";
import { parseLogLine, resetLogSeq } from "./lib/parseLog";
import type { Container, LogLine } from "./types";

export default function App() {
  const [containers, setContainers] = useState<Container[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [containerQuery, setContainerQuery] = useState("");
  const [logQuery, setLogQuery] = useState("");
  const [runningOnly, setRunningOnly] = useState(true);
  const [lines, setLines] = useState<LogLine[]>([]);
  const [dockerOk, setDockerOk] = useState<boolean | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [follow, setFollow] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);

  const selected = useMemo(
    () => containers.find((c) => c.id === selectedId) ?? null,
    [containers, selectedId],
  );

  const filteredContainers = useMemo(() => {
    const q = containerQuery.trim().toLowerCase();
    return containers.filter((c) => {
      if (runningOnly && c.state !== "running") return false;
      if (!q) return true;
      return (
        c.name.toLowerCase().includes(q) ||
        c.image.toLowerCase().includes(q) ||
        c.id.toLowerCase().includes(q)
      );
    });
  }, [containers, containerQuery, runningOnly]);

  const filteredLines = useMemo(() => {
    const q = logQuery.trim().toLowerCase();
    if (!q) return lines;
    return lines.filter((l) => l.raw.toLowerCase().includes(q));
  }, [lines, logQuery]);

  useEffect(() => {
    let cancelled = false;

    async function refresh() {
      try {
        const health = await fetchHealth();
        if (!cancelled) setDockerOk(Boolean(health.ok));
        const list = await fetchContainers();
        if (cancelled) return;
        setContainers(list);
        setError(null);
        setSelectedId((prev) => {
          if (prev && list.some((c) => c.id === prev)) return prev;
          const firstRunning = list.find((c) => c.state === "running");
          return firstRunning?.id ?? list[0]?.id ?? null;
        });
      } catch (err) {
        if (!cancelled) {
          setDockerOk(false);
          setError(err instanceof Error ? err.message : String(err));
        }
      }
    }

    refresh();
    const timer = window.setInterval(refresh, 5000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    if (!selectedId) {
      setLines([]);
      return;
    }

    resetLogSeq();
    setLines([]);
    setError(null);

    const es = openLogStream(selectedId, { tail: "300", follow: true });
    es.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as { message?: string };
        if (typeof data.message !== "string") return;
        setLines((prev) => {
          const next = [...prev, parseLogLine(data.message!)];
          return next.length > 5000 ? next.slice(next.length - 5000) : next;
        });
      } catch {
        // ignore malformed frames
      }
    };
    es.addEventListener("error", (ev) => {
      // EventSource "error" fires on disconnect too; only surface server payloads.
      const msg = (ev as MessageEvent).data;
      if (typeof msg === "string" && msg) {
        try {
          const data = JSON.parse(msg) as { error?: string };
          if (data.error) setError(data.error);
        } catch {
          /* network blip */
        }
      }
    });

    return () => es.close();
  }, [selectedId]);

  useEffect(() => {
    if (!follow || !logRef.current) return;
    logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [filteredLines, follow]);

  return (
    <div className="app">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Logka home">
          <span className="brand-mark">Logka</span>
          <span className="brand-tag">docker logs</span>
        </a>
        <div className="status-pill" title="Docker Engine connectivity">
          <span className={`status-dot${dockerOk ? " ok" : ""}`} aria-hidden />
          {dockerOk === null ? "checking docker…" : dockerOk ? "docker connected" : "docker offline"}
        </div>
      </header>

      <div className="shell">
        <aside className="sidebar">
          <h2>Containers</h2>
          <div className="filter-row">
            <input
              value={containerQuery}
              onChange={(e) => setContainerQuery(e.target.value)}
              placeholder="Filter name, image, id"
              aria-label="Filter containers"
            />
            <button
              type="button"
              className="ghost-btn"
              aria-pressed={runningOnly}
              onClick={() => setRunningOnly((v) => !v)}
            >
              {runningOnly ? "Running only" : "All states"}
            </button>
          </div>
          <ul className="container-list">
            {filteredContainers.map((c) => (
              <li key={c.id}>
                <button
                  type="button"
                  className={`container-item${c.id === selectedId ? " active" : ""}`}
                  onClick={() => setSelectedId(c.id)}
                >
                  <span className="container-name">{c.name || c.id.slice(0, 12)}</span>
                  <span className="container-meta">
                    <span className={`state ${c.state}`}>{c.state}</span>
                    <span>{c.image}</span>
                  </span>
                </button>
              </li>
            ))}
            {filteredContainers.length === 0 && (
              <li className="empty" style={{ margin: 0, padding: "1rem" }}>
                No containers match.
              </li>
            )}
          </ul>
        </aside>

        <section className="stream" aria-live="polite">
          <div className="stream-head">
            <div className="stream-title">
              <h2>Live stream</h2>
              {selected ? (
                <>
                  <strong>{selected.name || selected.id.slice(0, 12)}</strong>
                  <span>
                    {selected.status} · {selected.id.slice(0, 12)}
                  </span>
                </>
              ) : (
                <strong>Select a container</strong>
              )}
            </div>
            <div className="stream-tools">
              <input
                value={logQuery}
                onChange={(e) => setLogQuery(e.target.value)}
                placeholder="Search logs"
                aria-label="Search logs"
                disabled={!selected}
              />
              <button
                type="button"
                className="ghost-btn"
                aria-pressed={follow}
                onClick={() => setFollow((v) => !v)}
                disabled={!selected}
              >
                {follow ? "Following" : "Paused"}
              </button>
              <button
                type="button"
                className="ghost-btn"
                onClick={() => {
                  resetLogSeq();
                  setLines([]);
                }}
                disabled={!selected}
              >
                Clear
              </button>
            </div>
          </div>

          {error && <div className="error-banner">{error}</div>}

          {!selected ? (
            <div className="empty">
              <strong>Logka is ready</strong>
              Pick a container to stream stdout and stderr in real time.
            </div>
          ) : (
            <div className="log-view" ref={logRef}>
              {filteredLines.length === 0 ? (
                <div className="empty" style={{ margin: "1rem" }}>
                  Waiting for log lines…
                </div>
              ) : (
                filteredLines.map((line) => (
                  <div
                    key={line.id}
                    className={`log-line${logQuery && line.raw.toLowerCase().includes(logQuery.trim().toLowerCase()) ? " hit" : ""}`}
                  >
                    <span className="log-ts">{formatTs(line.ts)}</span>
                    <span className="log-body">{highlight(line.body, logQuery)}</span>
                  </div>
                ))
              )}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function formatTs(ts?: string): string {
  if (!ts) return "········";
  try {
    const d = new Date(ts);
    return d.toLocaleTimeString(undefined, { hour12: false }) + "." + String(d.getMilliseconds()).padStart(3, "0");
  } catch {
    return ts.slice(11, 23);
  }
}

function highlight(text: string, query: string) {
  const q = query.trim();
  if (!q) return text;
  const idx = text.toLowerCase().indexOf(q.toLowerCase());
  if (idx < 0) return text;
  return (
    <>
      {text.slice(0, idx)}
      <mark>{text.slice(idx, idx + q.length)}</mark>
      {text.slice(idx + q.length)}
    </>
  );
}
