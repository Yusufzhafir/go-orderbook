"use client";
import * as React from "react";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { useTrades } from "@/hooks/trade/useTrade";
import TradeTicker from "@/components/trade/TradeTicker";
import { AddOrderForm, ModifyCancelForm } from "@/components/trade/OrderForm";

export default function Page() {
  const [symbol, setSymbol] = React.useState("ticker");
  const [mode, setMode] = React.useState<"subscribeMessage" | "queryParam">("subscribeMessage");
  const { trades, connected } = useTrades(symbol, mode);

  return (
    <div className="p-4 grid lg:grid-cols-3 gap-4">
      <div className="lg:col-span-2 space-y-4">
        <Card className="p-4 space-y-3">
          <div className="font-semibold">Data Feed</div>
          <Separator />
          <div className="grid grid-cols-3 gap-3">
            <div>
              <Label>Symbol</Label>
              <Input value={symbol} onChange={(e) => setSymbol(e.target.value)} />
            </div>
            <div>
              <Label>WS Mode</Label>
              <Select value={mode} onValueChange={(v) => setMode(v as any)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="subscribeMessage">Send subscribe message</SelectItem>
                  <SelectItem value="queryParam">Use query param</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-end">
              <div className={`text-sm ${connected ? "text-green-600" : "text-red-600"}`}>
                {connected ? "Connected" : "Disconnected"}
              </div>
            </div>
          </div>
        </Card>

        <AddOrderForm />
        <ModifyCancelForm />
      </div>

      <div className="lg:col-span-1">
        <div className="h-[70vh]">
          <TradeTicker data={trades || []} />
        </div>
      </div>
    </div>
  );
}
