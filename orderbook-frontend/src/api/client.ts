// src/lib/api.ts
"use client";

import { getAccessToken } from "@/lib/token";

export const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

export enum Side {
  BID,
  ASK,
}

export type OrderTypeString = "LIMIT" | "MARKET";
export enum OrderType {
  ORDER_FILL_AND_KILL,
  ORDER_GOOD_TILL_CANCEL,
}

export function MapOrderType(orderType: OrderTypeString) {
  switch (orderType) {
    case "LIMIT":
      return OrderType.ORDER_GOOD_TILL_CANCEL;
    case "MARKET":
      return OrderType.ORDER_FILL_AND_KILL;
    default:
      return OrderType.ORDER_FILL_AND_KILL;
  }
}

export interface MatchorderType {
  Side: number;
  MakerID: number;
  TakerID: number;
  Price: number;
  Quantity: number;
  Timestamp: string;
}

export interface AddOrderRequest {
  side: Side;
  price: number; // uint64 server-side (send in smallest unit)
  quantity: number; // uint64 server-side
  type: OrderType;
  ticker: string
}
export interface AddOrderResponse {
  orderId: number;
  trades?: MatchorderType[];
  status: "accepted" | "rejected";
  message?: string;
}

export interface ModifyOrderRequest {
  id: number;
  price: number;
  quantity: number;
  type: OrderType;
  ticker: string
}

export type ModifyOrderResponse = AddOrderResponse;

export interface CancelOrderRequest {
  id: number;
}
export interface CancelOrderResponse {
  orderId: number;
  status: "accepted" | "rejected";
  message?: string;
}

export interface OrderBookLevel {
  price: number;
  volume: number;
  orderCount: number;
}
export interface OrderBookResponse {
  bids: OrderBookLevel[];
  asks: OrderBookLevel[];
  timestamp: number;
}

/** User & Auth types */
export interface UserProfile {
  id: number;
  username: string;
  created_at: string;
  balance: number
}
export interface LoginRequest {
  username: string;
  password: string;
}
export interface LoginResponse {
  token: string; // JWT
  user: UserProfile;
}
export interface RegisterRequest {
  username: string;
  password: string;
}
export interface RegisterResponse {
  userId: number;
}
export interface AddUserMoneyRequest {
  amount: number;
}
export interface AddUserMoneyResponse {
  message?: string;
}

export interface TickerListResponse {
  tickers: Ticker[]
}

interface Ticker {
  id: number
  ticker: string
}


export interface UserOrder {
  ID: number
  UserID: number
  TickerID: number
  Ticker: string,
  Side: Side
  TickerLedgerID: number
  Type: OrderType
  Quantity: number
  Filled: number
  Price: number
  IsActive: boolean
  CreatedAt: string
  ClosedAt: string
}

export type UserOrderList = UserOrder[]

async function http<T>(path: string, init: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init.headers as Record<string, string>),
  };

  // Attach JWT if present (all protected routes use authmiddleware)
  const token = getAccessToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  order: {
    addOrder: (b: AddOrderRequest) =>
      http<AddOrderResponse>("/api/v1/order/add", {
        method: "POST",
        body: JSON.stringify(b),
      }),
    modifyOrder: (b: ModifyOrderRequest) =>
      http<ModifyOrderResponse>("/api/v1/order/modify", {
        method: "PUT",
        body: JSON.stringify(b),
      }),
    cancelOrder: (b: CancelOrderRequest) =>
      http<CancelOrderResponse>("/api/v1/order/cancel", {
        method: "DELETE",
        body: JSON.stringify(b),
      }),
    fetchOrderBook: (ticker: string) =>
      http<OrderBookResponse>(`/api/v1/ticker/${ticker}/order-list`, {
        method: "GET",
      }),
  },
  ticker: {
    tickerList: () => http<TickerListResponse>("/api/v1/ticker", {
      method: "GET"
    })
  },
  user: {
    /** Public routes (no JWT) */
    register: (b: RegisterRequest) =>
      fetch(`${API_BASE}/api/v1/user/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(b),
      }).then(async (r) => {
        if (!r.ok) throw new Error((await r.text()) || `HTTP ${r.status}`);
        return (await r.json()) as RegisterResponse;
      }),
    login: (b: LoginRequest) =>
      fetch(`${API_BASE}/api/v1/user/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(b),
      }).then(async (r) => {
        if (!r.ok) throw new Error((await r.text()) || `HTTP ${r.status}`);
        return (await r.json()) as LoginResponse;
      }),

    /** Protected routes (JWT required) */
    getMe: () =>
      http<UserProfile>("/api/v1/user", { method: "GET" }),
    getMyOrders: () =>
      http<UserOrderList>("/api/v1/user/order-list", {
        method: "GET",
      }),
    addMoney: (b: AddUserMoneyRequest) =>
      http<AddUserMoneyResponse>("/api/v1/user/money", {
        method: "POST",
        body: JSON.stringify(b),
      }),
  },
};
