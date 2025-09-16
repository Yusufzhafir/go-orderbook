"use client"

import { useMe } from "@/hooks/auth/useAuth";
import { Button } from "../ui/button";
import { Github } from "lucide-react";
import LoginDialog from "./LoginDialog";
import RegisterDialog from "./registerDialog";
import AddMoneyDialog from "./AddMoneyDialog";

const Navbar = () => {
    const { data } = useMe()
    return (
        <div className="flex w-full bg-zinc-200 p-4 rounded-b-2xl shadow place-content-between">
            <div className="flex items-center gap-4">
                <h1 className="font-bold">
                    This is an Orderbook app written by
                    <span className="text-blue-500 cursor-pointer">
                        <a href="https://www.linkedin.com/in/yusuf-shadiq-034a741a5/" target="_blank" rel="noopener noreferrer">
                            {" "}me!
                        </a>
                    </span>
                </h1>
                <a href={"https://github.com/Yusufzhafir/go-orderbook"} target="_blank" rel="noopener noreferrer">
                    <Button>
                        Source Code
                        <span>
                            <Github />
                        </span>
                    </Button>
                </a>

            </div>
            <div className="flex gap-4 items-center">
                <div className="font-bold capitalize text-md">
                    name : <span className="font-medium text-sm">{data?.username}</span>
                </div>
                <div className="font-bold capitalize text-md">
                    balance : <span className="font-medium text-sm">{data?.balance.toLocaleString("us", {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2
                    })} USD</span>
                </div>
                <AddMoneyDialog />
                <LoginDialog />
                <RegisterDialog />
            </div>
        </div>
    );
}

export default Navbar;