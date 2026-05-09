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

    // onData/onResize 用 ref 中转，避免 stale closure：
    // useEffect 只依赖 readOnly，xterm 的 onData/onResize 注册一次即固化为对 ref 的解引用，
    // 之后父组件每次 re-render 传新回调（例如 ExperimentTerminal 里 ready 由 false→true 后
    // 重建的 handleTerminalData），都通过 ref 同步生效，不需要重建终端实例。
    // 直接把 onData/onResize 加到 useEffect 依赖会导致每次 re-render 都重建终端，丢失
    // 已有输出与滚动历史，是不可接受的。
    const onDataRef = useRef(onData);
    const onResizeRef = useRef(onResize);
    useEffect(() => { onDataRef.current = onData; }, [onData]);
    useEffect(() => { onResizeRef.current = onResize; }, [onResize]);

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

      if (!readOnly) {
        terminal.onData((data) => onDataRef.current?.(data));
      }
      terminal.onResize(({ cols, rows }) => onResizeRef.current?.(cols, rows));

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
