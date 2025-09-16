"use client";
import * as React from "react";
import { useTrades } from "@/hooks/trade/useTrade";
import { useOrderBook } from "@/hooks/trade/useOrderbook";
import OrderBook from "@/components/trade/OrderBook";
import TradeTicker from "@/components/trade/TradeTicker";
import { AddOrderForm } from "@/components/trade/OrderForm";
import Navbar from "@/components/layout/Navbar";
import OrderList from "@/components/trade/OrderList";
import TickerList from "@/components/trade/TickerList";
import { useSearchParams } from "next/navigation";

export default function Page() {
  const search = useSearchParams();
  const symbol = search.get("symbol") || "BBCAUSD";
  const [mode] = React.useState<"subscribeMessage" | "queryParam">("subscribeMessage");
  const { trades } = useTrades(symbol, mode); // optional
  const { data: book, error } = useOrderBook(symbol, 500);


  return (
    <div className="max-h-lvh overflow-y-hidden overflow-x-hidden ">
      <Navbar/>
      <div className="p-4 grid lg:grid-cols-3 gap-4">
        <div className="flex flex-col gap-4 lg:col-span-2">
          <div className="grid lg:grid-cols-2 gap-4 h-min">
            <div className="flex flex-col gap-4 lg:col-span-1">
              <TickerList/>
              <AddOrderForm ticker={symbol} />
            </div>
            <div className="lg:col-span-1">
              <OrderList ticker={""}/>
            </div>
          </div>
          <div className="h-[40vh]">
            <OrderBook bids={book?.bids ?? []} asks={book?.asks ?? []} depth={20} error={error} />
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
    </div>
  );
}