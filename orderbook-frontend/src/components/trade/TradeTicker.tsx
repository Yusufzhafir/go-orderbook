"use client";
import * as React from "react";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { TradeMsg } from "@/hooks/trade/useTrade";

type Row = { time: string; price: number; qty: number, symbol: string };
//   symbol: string;
//   price: number;
//   quantity: number;
//   timestamp?: number;
export default function TradeTicker({ data }: { data: TradeMsg[] }) {
    const rows: Row[] = React.useMemo(() => {
        return data.map((t) => ({
            price: t.price,
            qty: t.qty,
            symbol: t.symbol,
            time: t.ts ? new Date(t.ts).toLocaleTimeString() : "--:--:--",
        }))
    },[data])
    console.log(rows)


    return (
        <Card className="p-3 h-full overflow-auto">
            <div className="font-semibold mb-2">Running Trade</div>
            <Separator className="mb-2" />
            <div className="grid grid-cols-3 text-xs font-medium mb-2 opacity-70">
                <div>Time</div><div className="text-right">Price</div><div className="text-right">Qty</div>
            </div>
            <div className="space-y-1">
                {rows.map((r, i) => (
                    <div key={i} className="grid grid-cols-3 text-sm">
                        <div className="tabular-nums">{r.time}</div>
                        <div className="text-right tabular-nums">{r.price}</div>
                        <div className="text-right tabular-nums">{r.qty}</div>
                    </div>
                ))}
                {!rows.length && <div className="text-sm opacity-60">No trades yet…</div>}
            </div>
        </Card>
    );
}
