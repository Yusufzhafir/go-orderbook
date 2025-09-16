"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { setAccessToken, clearAccessToken } from "@/lib/token";
import { api, LoginRequest, RegisterRequest } from "@/api/client";

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: LoginRequest) => api.user.login(b),
    onSuccess: (res) => {
      setAccessToken(res.token);
      qc.invalidateQueries({ queryKey: ["me"] });
    },
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: (b: RegisterRequest) => api.user.register(b),
  });
}

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api.user.getMe(),
    retry: false,
    refetchInterval: 5000,
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return () => {
    clearAccessToken();
    qc.clear(); // wipe all cached queries
  };
}
