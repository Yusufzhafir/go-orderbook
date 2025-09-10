"use client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";

export default function Providers({ children }: { children: React.ReactNode }) {
  const [client] = React.useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { refetchOnWindowFocus: false, retry: 1 },
          mutations: { retry: 1 },
        },
      })
  );
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
