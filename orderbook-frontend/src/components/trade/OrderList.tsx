"use client";

import { OrderType, Side, UserOrder } from "@/api/client";
import { useCancelOrder, useMyOrders } from "@/hooks/trade/useOrderbook";
import { ScrollArea } from "@radix-ui/react-scroll-area";
import { useMemo } from "react";

function sideLabel(s: number) {
  return s === Side.BID ? "BUY" : "SELL";
}

function typeLabel(t: number) {
  switch (t) {
    case OrderType.ORDER_GOOD_TILL_CANCEL:
      return "LIMIT (GTC)";
    case OrderType.ORDER_FILL_AND_KILL:
      return "MARKET (IOC)";
    default:
      return String(t);
  }
}
function statusLabel(active?: boolean, closed_at?: string | null) {
  if (active === false) return "CLOSED";
  if (closed_at) return "CLOSED";
  return "OPEN";
}

const emptyArray: never[] = []
export default function OrderList({ ticker }: { ticker?: string }) {
  const { data, isLoading, error } = useMyOrders();
  const cancel = useCancelOrder();

  const orders: UserOrder[] = data || emptyArray;

  const filtered = useMemo(() => {
    console.log(orders)
    if (!ticker) return orders;
    // Try to match by 'ticker' string if available, otherwise leave as-is.
    return orders.filter((o) =>
      typeof o.Ticker === "string"
        ? o.Ticker.toUpperCase() === ticker.toUpperCase()
        : true
    );
  }, [orders, ticker]);

  return (
    <div className="border rounded-xl p-4 overflow-x-auto">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold">My Orders{ticker ? ` — ${ticker}` : ""}</h3>
        {isLoading && <span className="text-xs text-gray-500">Loading…</span>}
      </div>

      {error != null ? (
        <div className="text-sm text-red-600 mb-2 h-96">
          {String(error)}
        </div>
      ) : filtered.length === 0 ? (
        <div className="h-96">
          <p className="text-sm text-gray-500">No orders found.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <ScrollArea className="h-96">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left border-b">
                  <th className="py-2 pr-2">ID</th>
                  <th className="py-2 pr-2">Side</th>
                  <th className="py-2 pr-2">Type</th>
                  <th className="py-2 pr-2">Price</th>
                  <th className="py-2 pr-2">Qty</th>
                  <th className="py-2 pr-2">Status</th>
                  <th className="py-2 pr-2 w-24"></th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((o) => (
                  <tr key={o.ID} className="border-b last:border-0">
                    <td className="py-2 pr-2">{o.ID}</td>
                    <td className="py-2 pr-2">
                      <span
                        className={`px-2 py-0.5 rounded text-xs ${o.Side === Side.BID
                          ? "bg-green-100 text-green-700"
                          : "bg-red-100 text-red-700"
                          }`}
                      >
                        {sideLabel(o.Side)}
                      </span>
                    </td>
                    <td className="py-2 pr-2">{typeLabel(o.Type)}</td>
                    <td className="py-2 pr-2">{o.Price}</td>
                    <td className="py-2 pr-2">{o.Quantity}</td>
                    <td className="py-2 pr-2">
                      <span className="text-xs text-gray-600">
                        {statusLabel(o.IsActive, o.ClosedAt)}
                      </span>
                    </td>
                    <td className="py-2 pr-2">
                      <button
                        onClick={() => cancel.mutate({ id: o.ID })}
                        disabled={cancel.isPending || o.IsActive === false}
                        className={`px-2 py-1 rounded text-xs ${o.IsActive === false
                          ? "bg-gray-200 text-gray-500 cursor-not-allowed"
                          : "bg-red-600 text-white hover:bg-red-700"
                          }`}
                        title="Cancel order"
                      >
                        {cancel.isPending ? "…" : "Cancel"}
                      </button>
                    </td>
                  </tr>
                ))}

              </tbody>
            </table>
          </ScrollArea>

          {cancel.isError && (
            <p className="text-xs text-red-600 mt-2">{String(cancel.error)}</p>
          )}
        </div>
      )
      }
    </div>
  );
}
