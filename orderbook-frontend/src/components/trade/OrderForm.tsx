"use client";
import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import { api, MapOrderType, ModifyOrderRequest, OrderTypeString, Side } from "@/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

export function AddOrderForm({ className,ticker }: { className?: string,ticker:string }) {
  const [side, setSide] = React.useState<"BID" | "ASK">("BID");
  const [type, setType] = React.useState<OrderTypeString>("LIMIT");
  const [price, setPrice] = React.useState<string>("0");
  const [qty, setQty] = React.useState<string>("1");
  const orderCost = React.useMemo(()=>{
    const total = Number.parseInt(price)*Number.parseInt(qty)
    if (Number.isNaN(total)) {
      return 0
    }
    return total
  },[price,qty])
  const m = useMutation({ mutationFn: api.order.addOrder, onSuccess: (r) => alert(`${r.status}: ${r.orderId} ${r.message ?? ""}`), onError: (e) => alert(e.message) });

  const submit = React.useCallback(() => {
    m.mutate({
      side: side == "BID" ? Side.BID : Side.ASK,
      type: MapOrderType(type),
      price: Number(price),
      quantity: Number(qty),
      ticker : ticker
    });
  },[ticker,type,price,qty])

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
              <SelectItem value="IOC">IOC / FAK</SelectItem>
              <SelectItem value="FOK">FOK</SelectItem>
              <SelectItem value="GTC">GTC</SelectItem>
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
      <Button className="w-full" onClick={submit} disabled={m.isPending}>
        {m.isPending ? "Placing…" : "Place Order"}
      </Button>
    </Card>
  );
}

// export function ModifyCancelForm() {
//   const [id, setId] = React.useState<string>("");
//   const [price, setPrice] = React.useState<string>("");
//   const [qty, setQty] = React.useState<string>("");
//   const [type, setType] = React.useState<OrderTypeString | "">("");

//   const mModify = useMutation({
//     mutationFn: api.order.modifyOrder,
//     onSuccess: (r) => alert(`${r.status}: ${r.orderId} ${r.message ?? ""}`),
//     onError: (e) => alert(e.message),
//   });
//   const mCancel = useMutation({
//     mutationFn: api.order.cancelOrder,
//     onSuccess: (r) => alert(`${r.status}: ${r.orderId} ${r.message ?? ""}`),
//     onError: (e) => alert(e.message),
//   });

//   const doModify = () => {
//     if (!id) return alert("Order ID required");
//     const body: ModifyOrderRequest = {
//       id: Number(id),
//     };
//     if (price !== "") body.price = Number(price);
//     if (qty !== "") body.quantity = Number(qty);
//     if (type !== "") body.type = MapOrderType(type);
//     mModify.mutate(body);
//   };
//   const doCancel = () => {
//     if (!id) return alert("Order ID required");
//     mCancel.mutate({ id: Number(id) });
//   };

//   return (
//     <Card className="p-4 space-y-3">
//       <div className="font-semibold">Modify / Cancel</div>
//       <div className="grid grid-cols-3 gap-3">
//         <div>
//           <Label>Order ID</Label>
//           <Input type="number" value={id} onChange={(e) => setId(e.target.value)} />
//         </div>
//         <div>
//           <Label>New Price (opt)</Label>
//           <Input type="number" value={price} onChange={(e) => setPrice(e.target.value)} />
//         </div>
//         <div>
//           <Label>New Qty (opt)</Label>
//           <Input type="number" value={qty} onChange={(e) => setQty(e.target.value)} />
//         </div>
//         <div>
//           <Label>New Type (opt)</Label>
//           <Select value={type || ""} onValueChange={(v) => setType(v as OrderTypeString)}>
//             <SelectTrigger><SelectValue placeholder="—" /></SelectTrigger>
//             <SelectContent>
//               <SelectItem value="LIMIT">LIMIT</SelectItem>
//               <SelectItem value="MARKET">MARKET</SelectItem>
//             </SelectContent>
//           </Select>
//         </div>
//       </div>
//       <div className="flex gap-2">
//         <Button onClick={doModify} disabled={mModify.isPending}>Modify</Button>
//         <Button variant="destructive" onClick={doCancel} disabled={mCancel.isPending}>Cancel</Button>
//       </div>
//     </Card>
//   );
// }
