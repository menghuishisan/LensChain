'use client';

// XTermTerminal.tsx
// 基于 xterm.js 的 Web 终端组件
// 支持真 PTY 交互、ANSI 转义序列、光标控制、复制粘贴

import { useEffect, useRef, useImperativeHandle, forwardRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';

interface XTermTerminalProps {
  readOnly?: boolean;
  className?: string;
  onData?: (data: string) => void;
  onResize?: (cols: number, rows: number) => void;
}

export interface XTermTerminalHandle {
  write: (data: string) => void;
  clear: () => void;
  focus: () => void;
  fit: () => void;
}

/**
 * xterm.js 终端底层组件
 * 提供完整的终端仿真能力，支持 ANSI 转义、光标移动、复制粘贴
 */
export const XTermTerminal = forwardRef<XTermTerminalHandle, XTermTerminalProps>(
  function XTermTerminal({ readOnly = false, className, onData, onResize }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const terminalRef = useRef<Terminal | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);

    useImperativeHandle(ref, () => ({
      write: (data: string) => terminalRef.current?.write(data),
      clear: () => terminalRef.current?.clear(),
      focus: () => terminalRef.current?.focus(),
      fit: () => fitAddonRef.current?.fit(),
    }));

    useEffect(() => {
      if (!containerRef.current) return;

      const fitAddon = new FitAddon();
      const webLinksAddon = new WebLinksAddon();

      const terminal = new Terminal({
        cursorBlink: !readOnly,
        disableStdin: readOnly,
        fontSize: 14,
        fontFamily: "'Cascadia Code', 'Fira Code', 'Consolas', monospace",
        theme: {
          background: '#1e1e2e',
          foreground: '#cdd6f4',
          cursor: '#f5e0dc',
          selectionBackground: '#585b7066',
          black: '#45475a',
          red: '#f38ba8',
          green: '#a6e3a1',
          yellow: '#f9e2af',
          blue: '#89b4fa',
          magenta: '#f5c2e7',
          cyan: '#94e2d5',
          white: '#bac2de',
        },
        scrollback: 5000,
        convertEol: true,
      });

      terminal.loadAddon(fitAddon);
      terminal.loadAddon(webLinksAddon);
      terminal.open(containerRef.current);

      requestAnimationFrame(() => fitAddon.fit());

      fitAddonRef.current = fitAddon;
      terminalRef.current = terminal;

      if (!readOnly && onData) {
        terminal.onData(onData);
      }

      if (onResize) {
        terminal.onResize(({ cols, rows }) => onResize(cols, rows));
      }

      const resizeObserver = new ResizeObserver(() => {
        requestAnimationFrame(() => fitAddonRef.current?.fit());
      });
      resizeObserver.observe(containerRef.current);

      return () => {
        resizeObserver.disconnect();
        terminal.dispose();
        terminalRef.current = null;
        fitAddonRef.current = null;
      };
    }, [readOnly]);

    return <div ref={containerRef} className={`min-h-[300px] ${className ?? ''}`} />;
  }
);
