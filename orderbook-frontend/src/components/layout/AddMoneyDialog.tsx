"use client"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCallback, useState } from "react";
import { useAddMoney } from "@/hooks/auth/useAuth";
import { LoadingSpinner } from "@/components/ui/loadingSpinner";

const AddMoneyDialog = () => {
    const [open, setOpen] = useState(false)
    const [amount, setAmount] = useState(0)
    const { mutateAsync, isPending } = useAddMoney()
    const submit = useCallback(async () => {
        try {
            const result = await mutateAsync({
                amount :amount
            })
            console.log(result)
        } catch (e) {
            alert("AddMoney failed" + JSON.stringify(e))
        } finally {
            setOpen(false)
        }
    }, [amount, mutateAsync, setOpen])
    return (
        <Dialog open={open} onOpenChange={(v) => {
            setOpen(v)
        }}>
            <DialogTrigger asChild>
                <Button>
                    Add Money
                </Button>
            </DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Add Money</DialogTitle>
                    <form onSubmit={async (e) => {
                        e.preventDefault()
                        await submit()
                    }}
                        className="flex flex-col gap-2"
                    >
                        <Label htmlFor="amount">add money amount (USD)</Label>
                        <Input name="amount" type="number" onChange={(e) => setAmount(e.target.valueAsNumber)} />
                        <Button type="submit" disabled={isPending}>
                            {!isPending ? "Submit" : (
                                <LoadingSpinner />
                            )}
                        </Button>
                    </form>
                </DialogHeader>
            </DialogContent>
        </Dialog>
    );
}

export default AddMoneyDialog;