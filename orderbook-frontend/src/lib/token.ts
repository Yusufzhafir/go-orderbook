let accessToken: string | null = null;

export function setAccessToken(t: string | null) {
  accessToken = t;
  if (typeof window !== "undefined") {
    if (t) localStorage.setItem("jwt", t);
    else localStorage.removeItem("jwt");
  }
}

export function getAccessToken(): string | null {
  if (accessToken) return accessToken;
  if (typeof window !== "undefined") {
    const t = localStorage.getItem("jwt");
    accessToken = t || null;
    return accessToken;
  }
  return null;
}

export function clearAccessToken() {
  setAccessToken(null);
}
