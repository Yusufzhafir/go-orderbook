"use client"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCallback, useState } from "react";
import { useRegister } from "@/hooks/auth/useAuth";
import { LoadingSpinner } from "@/components/ui/loadingSpinner";

const RegisterDialog = () => {
    const [open, setOpen] = useState(false)
    const [username, setUsername] = useState("")
    const [password, setPassword] = useState("")
    const { mutateAsync, isPending } = useRegister()
    const submit = useCallback(async () => {
        try {
            const result = await mutateAsync({
                password: password,
                username: username
            })
            console.log(result)
        } catch (e) {
            alert("Register failed" + JSON.stringify(e))
        } finally {
            setOpen(false)
        }
    }, [username, password, mutateAsync, setOpen])
    return (
        <Dialog open={open} onOpenChange={(v) => {
            setOpen(v)
        }}>
            <DialogTrigger asChild>
                <Button>
                    Register
                </Button>
            </DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Register</DialogTitle>
                    <form onSubmit={async (e) => {
                        e.preventDefault()
                        await submit()
                    }}
                        className="flex flex-col gap-2"
                    >
                        <Label htmlFor="username">username</Label>
                        <Input name="username" onChange={(e) => setUsername(e.target.value)} />
                        <Label htmlFor="password">password</Label>
                        <Input name="password" onChange={(e) => setPassword(e.target.value)} />
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

export default RegisterDialog;