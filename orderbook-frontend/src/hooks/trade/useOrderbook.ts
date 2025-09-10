"use client";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api/client";

export function useOrderBook(symbol: string, depth = 20) {
  return useQuery({
    queryKey: ["orderbook", symbol, depth],
    queryFn: () => api.fetchOrderBook(symbol),
    // Polling keeps it simple; tune as you like
    refetchInterval: 700, // ms
    refetchOnWindowFocus: false,
  });
}
