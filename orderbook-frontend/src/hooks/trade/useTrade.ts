"use client";
import * as React from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

export type TradeMsg = {
  symbol: string,
  price: number,
  qty: number,
  side: string,
  ts: number,
  seq: number
};

export type TradeMsgBody = {
  type: "trade",
  trade: TradeMsg
}

type Mode = "subscribeMessage" | "queryParam";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";

export function useTrades(symbol: string, mode: Mode = "subscribeMessage") {
  const qc = useQueryClient();
  const [connected, setConnected] = React.useState(false);

  // Holder query: we never fetch; WS will populate it
  const { data } = useQuery<TradeMsg[] | undefined>({
    queryKey: ["trades", symbol],
    queryFn: async () => [],
    initialData: [],
    staleTime: Infinity,
    gcTime: Infinity,
  });

  React.useEffect(() => {
    // reset stream buffer for symbol
    qc.setQueryData<TradeMsg[]>(["trades", symbol], []);

    const url =
      mode === "queryParam" ? `${WS_URL}?symbols=${encodeURIComponent(symbol)}` : WS_URL;

    const ws = new WebSocket(url);

    ws.onopen = () => {
      setConnected(true);
      if (mode === "subscribeMessage") {
        ws.send(JSON.stringify({ type: "subscribe", symbol }));
      }
    };

    ws.onmessage = (ev) => {
      try {
        const payload = ev.data as string
        const parsePayload = payload.split("\n").map((e)=>(JSON.parse(e) as TradeMsgBody).trade)

        qc.setQueryData<TradeMsg[]>(["trades", symbol], (prev) => {
          const curr = prev || []
          const next = [...parsePayload, ...curr].slice(0, 300);
          return next;
        });
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    return () => ws.close();
  }, [symbol, mode, qc]);

  return { trades: data, connected };
}
