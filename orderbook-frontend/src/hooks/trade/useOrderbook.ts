// src/hooks/useOrderbook.ts
"use client";

import { AddOrderRequest, api, CancelOrderRequest, ModifyOrderRequest } from "@/api/client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useOrderBook(ticker: string, refetchMs = 1000) {
  return useQuery({
    queryKey: ["orderbook", ticker],
    queryFn: () => api.order.fetchOrderBook(ticker),
    refetchInterval: refetchMs,
  });
}

export function useMyOrders() {
  return useQuery({
    queryKey: ["my-orders"],
    queryFn: () => api.user.getMyOrders(),
    staleTime: 5_000,
  });
}

export function useAddOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: AddOrderRequest) => api.order.addOrder(b),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["orderbook"] });
      qc.invalidateQueries({ queryKey: ["my-orders"] });
    },
  });
}

export function useModifyOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: ModifyOrderRequest) => api.order.modifyOrder(b),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["orderbook"] });
      qc.invalidateQueries({ queryKey: ["my-orders"] });
    },
  });
}

export function useCancelOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: CancelOrderRequest) => api.order.cancelOrder(b),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["orderbook"] });
      qc.invalidateQueries({ queryKey: ["my-orders"] });
    },
  });
}
