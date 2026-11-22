// Strictly in-memory Access Token storage.
// Never persisted to localStorage or sessionStorage to protect against XSS token theft.
let inMemoryAccessToken: string | null = null;

export function getAccessToken(): string | null {
  return inMemoryAccessToken;
}

export function setAccessToken(token: string | null): void {
  inMemoryAccessToken = token;
}
