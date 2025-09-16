"use client";

import * as React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api/client";

export default function TickerList() {
  const router = useRouter();
  const search = useSearchParams();

  const { data, isLoading, error } = useQuery({
    queryKey: ["tickers"],
    queryFn: api.ticker.tickerList,
    staleTime: 30_000,
  });

  const currentSymbol = search.get("symbol") ?? "";

  // If no symbol in URL yet, default to the first ticker returned by the API
  React.useEffect(() => {
    if (!data?.tickers?.length) return;
    if (!currentSymbol) {
      const first = data.tickers[0]?.ticker;
      if (first) {
        const params = new URLSearchParams(Array.from(search.entries()));
        params.set("symbol", first);
        router.replace(`?${params.toString()}`, { scroll: false });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, currentSymbol]);

  const onChange = (nextSymbol: string) => {
    const params = new URLSearchParams(Array.from(search.entries()));
    params.set("symbol", nextSymbol);
    router.replace(`?${params.toString()}`, { scroll: false });
  };

  return (
    <div className="border rounded-xl p-3 flex items-center gap-2">
      <label htmlFor="ticker" className="text-sm text-gray-600">
        Ticker
      </label>

      {isLoading ? (
        <span className="text-xs text-gray-500">Loading…</span>
      ) : error ? (
        <span className="text-xs text-red-600">Failed to load tickers</span>
      ) : (
        <select
          id="ticker"
          value={currentSymbol}
          onChange={(e) => onChange(e.target.value)}
          className="px-2 py-1 border rounded w-full"
        >
          {(data?.tickers ?? []).map((t) => (
            <option key={t.id} value={t.ticker}>
              {t.ticker}
            </option>
          ))}
        </select>
      )}
    </div>
  );
}
