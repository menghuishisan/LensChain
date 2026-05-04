// xterm-server — Web 终端服务
// 基于 node-pty 提供真实 PTY 终端，通过 WebSocket 与前端 xterm.js 通信。
// 每个 WebSocket 连接分配独立的 PTY 会话，支持终端 resize。

import http from 'http';
import crypto from 'crypto';
import { WebSocketServer, WebSocket } from 'ws';
import { spawn, IPty } from 'node-pty';

const PORT = parseInt(process.env.PORT || '3000', 10);
const DEFAULT_SHELL = process.env.SHELL || '/bin/sh';
const DEFAULT_COLS = parseInt(process.env.COLS || '120', 10);
const DEFAULT_ROWS = parseInt(process.env.ROWS || '30', 10);
const AUTH_TOKEN = process.env.AUTH_TOKEN || '';
const MAX_SESSIONS = parseInt(process.env.MAX_SESSIONS || '64', 10);
const ALLOWED_SHELLS = (process.env.ALLOWED_SHELLS || '/bin/sh,/bin/bash,/bin/zsh,/bin/ash').split(',').map(s => s.trim()).filter(Boolean);
const WS_MAX_PAYLOAD = 64 * 1024; // 64 KB

interface Session {
  id: string;
  pty: IPty;
  ws: WebSocket;
  createdAt: number;
}

const sessions = new Map<string, Session>();

// ── HTTP 服务 ──────────────────────────────────────────────────

const server = http.createServer((req, res) => {
  // 健康检查：存活探针，始终返回 ok。
  if (req.url === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'ok',
      active_sessions: sessions.size,
      max_sessions: MAX_SESSIONS,
      uptime_seconds: Math.floor(process.uptime()),
    }));
    return;
  }

  // 就绪探针：当会话数未达上限时标记为就绪。
  if (req.url === '/readyz') {
    const ready = sessions.size < MAX_SESSIONS;
    res.writeHead(ready ? 200 : 503, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      ready,
      active_sessions: sessions.size,
      max_sessions: MAX_SESSIONS,
    }));
    return;
  }

  // 会话列表：返回当前活跃的终端会话信息（不含敏感数据）。
  if (req.url === '/api/v1/sessions') {
    if (AUTH_TOKEN) {
      const authHeader = req.headers['authorization'] || '';
      if (authHeader !== `Bearer ${AUTH_TOKEN}`) {
        res.writeHead(401, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'unauthorized' }));
        return;
      }
    }
    const list = Array.from(sessions.values()).map((s) => ({
      id: s.id,
      created_at: new Date(s.createdAt).toISOString(),
      age_seconds: Math.floor((Date.now() - s.createdAt) / 1000),
    }));
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ sessions: list, total: list.length }));
    return;
  }

  res.writeHead(404, { 'Content-Type': 'text/plain' });
  res.end('Not Found');
});

// ── WebSocket 服务 ─────────────────────────────────────────────

const wss = new WebSocketServer({ server, path: '/ws', maxPayload: WS_MAX_PAYLOAD });

wss.on('connection', (ws: WebSocket, req: http.IncomingMessage) => {
  // 认证检查
  if (AUTH_TOKEN) {
    const url = new URL(req.url || '/', `http://${req.headers.host}`);
    const token = url.searchParams.get('token') || '';
    if (token !== AUTH_TOKEN) {
      ws.close(4001, 'Unauthorized');
      return;
    }
  }

  if (sessions.size >= MAX_SESSIONS) {
    ws.close(4002, 'Too many sessions');
    return;
  }

  const url = new URL(req.url || '/', `http://${req.headers.host}`);
  const requestedShell = url.searchParams.get('shell') || DEFAULT_SHELL;
  const shell = ALLOWED_SHELLS.includes(requestedShell) ? requestedShell : DEFAULT_SHELL;
  const cols = clamp(parseInt(url.searchParams.get('cols') || String(DEFAULT_COLS), 10), 10, 500);
  const rows = clamp(parseInt(url.searchParams.get('rows') || String(DEFAULT_ROWS), 10), 2, 200);

  const sessionId = crypto.randomUUID();
  let pty: IPty;

  try {
    pty = spawn(shell, [], {
      name: 'xterm-256color',
      cols,
      rows,
      cwd: process.env.HOME || '/',
      env: {
        ...process.env,
        TERM: 'xterm-256color',
        COLORTERM: 'truecolor',
      } as Record<string, string>,
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    ws.close(4003, `Failed to spawn shell: ${msg}`);
    return;
  }

  const session: Session = { id: sessionId, pty, ws, createdAt: Date.now() };
  sessions.set(sessionId, session);

  // PTY → WebSocket
  pty.onData((data: string) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(data);
    }
  });

  pty.onExit(({ exitCode }) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.close(1000, `Shell exited with code ${exitCode}`);
    }
    sessions.delete(sessionId);
  });

  // WebSocket → PTY
  ws.on('message', (raw: Buffer | string) => {
    const msg = raw.toString();

    // JSON 控制消息（resize）
    if (msg.startsWith('{')) {
      try {
        const parsed = JSON.parse(msg);
        if (parsed.type === 'resize' && typeof parsed.cols === 'number' && typeof parsed.rows === 'number') {
          pty.resize(clamp(parsed.cols, 10, 500), clamp(parsed.rows, 2, 200));
          return;
        }
      } catch {
        // 非 JSON，当作普通输入
      }
    }

    pty.write(msg);
  });

  ws.on('close', () => cleanup(sessionId));
  ws.on('error', () => cleanup(sessionId));
});

function cleanup(sessionId: string): void {
  const session = sessions.get(sessionId);
  if (session) {
    try { session.pty.kill(); } catch { /* ignore */ }
    sessions.delete(sessionId);
  }
}

function clamp(value: number, min: number, max: number): number {
  if (isNaN(value)) return min;
  return Math.max(min, Math.min(max, value));
}

// ── 启动 ───────────────────────────────────────────────────────

server.listen(PORT, '0.0.0.0', () => {
  console.log(`xterm-server listening on 0.0.0.0:${PORT}`);
});

// ── 优雅退出 ───────────────────────────────────────────────────

function shutdown(): void {
  console.log('Shutting down...');
  sessions.forEach((session) => {
    try { session.pty.kill(); } catch { /* ignore */ }
    if (session.ws.readyState === WebSocket.OPEN) {
      session.ws.close(1001, 'Server shutting down');
    }
  });
  sessions.clear();
  server.close(() => process.exit(0));
  setTimeout(() => process.exit(1), 5000);
}

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);
