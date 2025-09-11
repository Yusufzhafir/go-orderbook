"use client";
import * as React from "react";
import { useTrades } from "@/hooks/trade/useTrade";
import { useOrderBook } from "@/hooks/trade/useOrderbook";
import OrderBook from "@/components/trade/OrderBook";
import TradeTicker from "@/components/trade/TradeTicker";
import { AddOrderForm, ModifyCancelForm } from "@/components/trade/OrderForm";

export default function Page() {
  const [symbol] = React.useState("ticker");
  const [mode] = React.useState<"subscribeMessage" | "queryParam">("subscribeMessage");
  const { trades } = useTrades(symbol, mode); // optional
  const { data: book, error } = useOrderBook(symbol, 20);

  return (
    <div className="p-4 grid lg:grid-cols-3 gap-4">
      <div className="flex flex-col gap-4 lg:col-span-2">
        <div className="grid lg:grid-cols-2 gap-4">
          <div className="lg:col-span-1">
            <AddOrderForm />
          </div>
          <div className="lg:col-span-1">
            <ModifyCancelForm />
          </div>
        </div>
        <div className="h-[60vh]">
          <OrderBook bids={book?.bids ?? []} asks={book?.asks ?? []} depth={20} error={error}/>
        </div>
      </div>
      <div className="lg:col-span-1">
        <div className="h-[70vh]">
          <TradeTicker data={trades || []} />
        </div>
      </div>
      <div className="lg:col-span-2 space-y-4">
      </div>

    </div>
  );
}
