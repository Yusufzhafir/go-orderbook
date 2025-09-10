const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

export enum Side {
  BID,
  ASK
}

export type OrderTypeString = "LIMIT" | "MARKET"
export enum OrderType {
  ORDER_FILL_AND_KILL,
  ORDER_GOOD_TILL_CANCEL
}

export function MapOrderType(orderType: OrderTypeString) {
  switch (orderType) {
    case "LIMIT":
      return OrderType.ORDER_GOOD_TILL_CANCEL
    case "MARKET":
      return OrderType.ORDER_FILL_AND_KILL
    default:
      return OrderType.ORDER_FILL_AND_KILL
  }
}

interface MatchorderType {
  Side: number
  MakerID: number
  TakerID: number
  Price: number
  Quantity: number
  Timestamp: string
}
export interface AddOrderRequest {
  side: Side;
  price: number;     // uint64 server-side
  quantity: number;  // uint64 server-side
  type: OrderType;
}

export interface AddOrderResponse {
  orderId: number;
  trades?: MatchorderType[];
  status: "accepted" | "rejected";
  message?: string;
}

export interface ModifyOrderRequest {
  id: number;
  price?: number;
  quantity?: number;
  type?: OrderType;
}
export type ModifyOrderResponse =  AddOrderResponse

export interface CancelOrderRequest { id: number; }
export interface CancelOrderResponse {
  orderId: number;
  status: "accepted" | "rejected";
  message?: string;
}

export interface OrderBookResponse {
  bids: OrderBookLevel[]
  asks: OrderBookLevel[]
  timestamp: number
}

export interface OrderBookLevel {
  price: number
  volume: number
  orderCount: number
}


async function http<T>(path: string, init: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init.headers || {}) },
  });
  if (!res.ok) throw new Error((await res.text()) || `HTTP ${res.status}`);
  return res.json() as Promise<T>;
}

export const api = {
  addOrder: (b: AddOrderRequest) =>
    http<AddOrderResponse>("/api/v1/order/add", { method: "POST", body: JSON.stringify(b) }),
  modifyOrder: (b: ModifyOrderRequest) =>
    http<ModifyOrderResponse>("/api/v1/order/modify", { method: "PUT", body: JSON.stringify(b) }),
  cancelOrder: (b: CancelOrderRequest) =>
    http<CancelOrderResponse>("/api/v1/order/cancel", { method: "DELETE", body: JSON.stringify(b) }),
  fetchOrderBook: (ticker: string) =>
    http<OrderBookResponse>(`/api/v1/ticker/${ticker}/order-list`, { method: "GET" }),
};
