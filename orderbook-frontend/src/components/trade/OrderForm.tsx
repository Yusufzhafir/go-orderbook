"use client";
import * as React from "react";
import { MapOrderType, OrderTypeString, Side } from "@/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { useAddOrder } from "@/hooks/trade/useOrderbook";

export function AddOrderForm({ className,ticker }: { className?: string,ticker:string }) {
  const [side, setSide] = React.useState<"BID" | "ASK">("BID");
  const [type, setType] = React.useState<OrderTypeString>("LIMIT");
  const [price, setPrice] = React.useState<string>("0");
  const [qty, setQty] = React.useState<string>("1");
  const {mutate,isPending} = useAddOrder()
  const orderCost = React.useMemo(()=>{
    const total = Number.parseInt(price)*Number.parseInt(qty)
    if (Number.isNaN(total)) {
      return 0
    }
    return total
  },[price,qty])

  const submit = React.useCallback(() => {
    mutate({
      side: side == "BID" ? Side.BID : Side.ASK,
      type: MapOrderType(type),
      price: Number(price),
      quantity: Number(qty),
      ticker : ticker
    });
  },[ticker,type,price,qty,side,mutate])

  return (
    <Card className={cn("p-4 space-y-3",className)}>
      <div className="font-semibold">Add Order</div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label>Side</Label>
          <Select value={side} onValueChange={(v) => setSide(v as "BID" | "ASK")}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="BID">BID (Buy)</SelectItem>
              <SelectItem value="ASK">ASK (Sell)</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <Label>Type</Label>
          <Select value={type} onValueChange={(v) => setType(v as OrderTypeString)}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="LIMIT">LIMIT</SelectItem>
              <SelectItem value="MARKET">MARKET</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-2">
          <Label> (USD)</Label>
          <Input type="number" inputMode="numeric" value={price} onChange={(e) => setPrice(e.target.value)} />
          <Label> cost {orderCost}</Label>
        </div>
        <div className="flex flex-col gap-2">
          <Label>Quantity</Label>
          <Input type="number" inputMode="numeric" value={qty} onChange={(e) => setQty(e.target.value)} />
        </div>
      </div>
      <Button className="w-full" onClick={submit} disabled={isPending}>
        {isPending ? "Placing…" : "Place Order"}
      </Button>
    </Card>
  );
}
