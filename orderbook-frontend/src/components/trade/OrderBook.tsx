"use client";
import * as React from "react";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils"; // optional; or inline classNames
import { type OrderBookLevel } from "@/api/client";

function fmt(n: number | undefined) {
    if (n == null || Number.isNaN(n)) return "-";
    return n.toLocaleString();
}

function DepthRow({
    level,
    maxQty,
    side,
}: {
    level: OrderBookLevel;
    maxQty: number;
    side: "bid" | "ask";
}) {
    const pct = maxQty > 0 ? Math.min(100, (level.volume / maxQty) * 100) : 0;
    const bg =
        side === "bid" ? "bg-emerald-500/20" : "bg-red-500/20";
    const priceColor =
        side === "bid" ? "text-emerald-600" : "text-red-600";

    return (
        <div className="relative grid grid-cols-3 text-sm px-2 py-0.5">
            <div
                className={cn("absolute inset-y-0", side === "bid" ? "right-0" : "left-0")}
                style={{
                    width: `${pct}%`,
                    backgroundColor: "transparent",
                }}
            >
                <div className={cn("h-full", bg)} />
            </div>
            <div className="z-10 tabular-nums">{fmt(level.volume)}</div>
            <div className={cn("z-10 tabular-nums text-right", priceColor)}>{fmt(level.price)}</div>
            <div className="z-10 tabular-nums text-right">{fmt(level.price * level.volume)}</div>
        </div>
    );
}

function sideMaxQty(levels: OrderBookLevel[]) {
    return levels.reduce((m, l) => Math.max(m, l.volume || 0), 0);
}

export default function OrderBook({
    bids,
    asks,
    depth = 20,
    error
}: {
    bids: OrderBookLevel[];
    asks: OrderBookLevel[];
    depth?: number;
    error: Error | null
}) {
    // Ensure sort (defensive if backend doesn’t sort)
    const asksSorted = React.useMemo(
        () => [...asks].sort((a, b) => a.price - b.price).slice(0, depth),
        [asks, depth]
    );
    const bidsSorted = React.useMemo(
        () => [...bids].sort((a, b) => b.price - a.price).slice(0, depth),
        [bids, depth]
    );

    const maxBidQty = sideMaxQty(bidsSorted);
    const maxAskQty = sideMaxQty(asksSorted);

    const bestBid = bidsSorted[0]?.price;
    const bestAsk = asksSorted[0]?.price;
    const spread = bestAsk && bestBid ? bestAsk - bestBid : undefined;

    return (
        <Card className="p-3 h-full">
            <div className="flex items-center justify-between">
                <div className="font-semibold">Order Book</div>
                <div className="text-xs opacity-70">
                    Spread: <span className="tabular-nums">{fmt(spread)}</span>
                </div>
            </div>

            {error != null && (
                <Card className="p-4 text-sm text-red-600">Failed to load order book: {(error as Error).message}</Card>
            )}

            <Separator className="my-2" />

            <div className="grid grid-cols-2 gap-3">
                {/* Bids (buy) */}
                <div className="rounded border p-2">
                    <div className="grid grid-cols-3 text-xs font-medium opacity-70 px-2">
                        <div>Size</div>
                        <div className="text-right">Price</div>
                        <div className="text-right">Total</div>
                    </div>
                    <div className="mt-1 space-y-0.5">
                        {bidsSorted.map((lvl, i) => (
                            <DepthRow key={`bid-${i}`} level={lvl} maxQty={maxBidQty} side="bid" />
                        ))}
                        {!bidsSorted.length && <div className="text-sm opacity-60 px-2 py-1">No bids</div>}
                    </div>
                </div>

                {/* Asks (sell) */}
                <div className="rounded border p-2">
                    <div className="grid grid-cols-3 text-xs font-medium opacity-70 px-2">
                        <div>Size</div>
                        <div className="text-right">Price</div>
                        <div className="text-right">Total</div>
                    </div>
                    <div className="mt-1 space-y-0.5">
                        {asksSorted.map((lvl, i) => (
                            <DepthRow key={`ask-${i}`} level={lvl} maxQty={maxAskQty} side="ask" />
                        ))}
                        {!asksSorted.length && <div className="text-sm opacity-60 px-2 py-1">No asks</div>}
                    </div>
                </div>
            </div>
        </Card>
    );
}
