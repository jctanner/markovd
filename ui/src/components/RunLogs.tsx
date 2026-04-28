import { useState, useEffect, useRef } from 'react';
import { api } from '../api';

interface Props {
  runID: string;
  status: string;
}

function parseSseLines(buffer: string): { lines: string[]; remainder: string } {
  const lines: string[] = [];
  let remainder = buffer;
  let idx: number;
  while ((idx = remainder.indexOf('\n\n')) !== -1) {
    const frame = remainder.slice(0, idx);
    remainder = remainder.slice(idx + 2);
    for (const line of frame.split('\n')) {
      if (line.startsWith('data: ')) {
        lines.push(line.slice(6));
      }
    }
  }
  return { lines, remainder };
}

export default function RunLogs({ runID, status }: Props) {
  const [logs, setLogs] = useState('');
  const [loading, setLoading] = useState(true);
  const [streaming, setStreaming] = useState(false);
  const [cached, setCached] = useState(false);
  const logsRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    setLogs('');
    setLoading(true);
    setCached(false);
    setStreaming(false);

    if (status === 'running' || status === 'pending') {
      const controller = new AbortController();
      let retryTimer: ReturnType<typeof setTimeout> | null = null;

      const connect = () => {
        const token = localStorage.getItem('token');

        fetch(`/api/v1/runs/${encodeURIComponent(runID)}/logs/stream`, {
          headers: { 'Authorization': `Bearer ${token}` },
          signal: controller.signal,
        }).then(async (res) => {
          if (!res.body) {
            setLogs('Streaming not supported');
            setLoading(false);
            return;
          }
          setLoading(false);
          setStreaming(true);

          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';
          let accumulated = '';

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const { lines, remainder } = parseSseLines(buffer);
            buffer = remainder;
            if (lines.length > 0) {
              accumulated += lines.join('\n') + '\n';
              setLogs(accumulated);
            }
          }
          setStreaming(false);
          if (!controller.signal.aborted) {
            retryTimer = setTimeout(connect, 3000);
          }
        }).catch((err) => {
          if (err.name === 'AbortError') return;
          setLoading(false);
          setStreaming(false);
          if (!controller.signal.aborted) {
            retryTimer = setTimeout(connect, 3000);
          }
        });
      };

      connect();

      return () => {
        controller.abort();
        if (retryTimer) clearTimeout(retryTimer);
      };
    }

    api.getRunLogs(runID).then((res) => {
      if (res.logs) {
        setLogs(res.logs);
        setCached(res.cached === 'true');
      } else {
        setLogs(res.error || 'No logs available');
      }
    }).catch(() => {
      setLogs('Failed to fetch logs');
    }).finally(() => setLoading(false));
  }, [runID, status]);

  useEffect(() => {
    if (streaming && logsRef.current) {
      logsRef.current.scrollTop = logsRef.current.scrollHeight;
    }
  }, [logs, streaming]);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <span className="meta-label" style={{ margin: 0 }}>Run Logs</span>
        {streaming && <span className="modal-live-badge">Live</span>}
        {cached && <span className="badge badge-pending">cached</span>}
      </div>
      {loading ? (
        <div className="loading-state">Loading logs...</div>
      ) : (
        <pre className="run-logs" ref={logsRef}>{logs}</pre>
      )}
    </div>
  );
}
