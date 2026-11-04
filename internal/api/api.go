package api

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/debelistic/logka/internal/docker"
)

// Register mounts JSON and SSE endpoints on mux.
func Register(mux *http.ServeMux, client *docker.Client) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]any{"ok": true, "service": "logka"}
		if err := client.Ping(); err != nil {
			status["ok"] = false
			status["docker"] = err.Error()
			writeJSON(w, http.StatusServiceUnavailable, status)
			return
		}
		status["docker"] = "ok"
		writeJSON(w, http.StatusOK, status)
	})

	mux.HandleFunc("GET /api/containers", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		list, err := client.ListContainers(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"containers": list})
	})

	mux.HandleFunc("GET /api/containers/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("missing container id"))
			return
		}
		follow := r.URL.Query().Get("follow") != "false"
		tail := r.URL.Query().Get("tail")
		if tail == "" {
			tail = "200"
		}
		streamLogs(w, r, client, id, follow, tail)
	})
}

func streamLogs(w http.ResponseWriter, r *http.Request, client *docker.Client, id string, follow bool, tail string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming unsupported"))
		return
	}

	tty, err := client.InspectTty(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	rc, err := client.StreamLogs(r.Context(), id, follow, tail)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	ctx := r.Context()
	lines := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(lines)
		if tty {
			errCh <- streamPlain(ctx, rc, lines)
			return
		}
		errCh <- streamMultiplexed(ctx, rc, lines)
	}()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case line, ok := <-lines:
			if !ok {
				select {
				case err := <-errCh:
					if err != nil && ctx.Err() == nil && err != io.EOF {
						payload, _ := json.Marshal(map[string]string{"error": err.Error()})
						fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
						flusher.Flush()
					}
				default:
				}
				return
			}
			payload, _ := json.Marshal(map[string]string{"message": line})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

func streamPlain(ctx context.Context, r io.Reader, out chan<- string) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- scanner.Text():
		}
	}
	return scanner.Err()
}

// Docker non-TTY logs are framed: 8-byte header + payload.
func streamMultiplexed(ctx context.Context, r io.Reader, out chan<- string) error {
	hdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, hdr); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return io.EOF
			}
			return err
		}
		size := binary.BigEndian.Uint32(hdr[4:])
		if size == 0 {
			continue
		}
		buf := make([]byte, size)
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		text := strings.TrimRight(string(buf), "\r\n")
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimRight(line, "\r")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- line:
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
