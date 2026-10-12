import type { NewTicket, Page, Stats, Status, Ticket } from "./types";

export class ApiError extends Error {
  constructor(
    message: string,
    readonly fields: Record<string, string> = {},
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(body?.error?.message ?? `Request failed (${res.status})`, body?.error?.fields);
  }
  return body as T;
}

export const api = {
  listTickets(params: { status?: Status; page: number; limit: number }) {
    const query = new URLSearchParams({ page: String(params.page), limit: String(params.limit) });
    if (params.status) query.set("status", params.status);
    return request<Page<Ticket>>(`/tickets?${query}`);
  },
  createTicket(ticket: NewTicket) {
    return request<Ticket>("/tickets", { method: "POST", body: JSON.stringify(ticket) });
  },
  updateTicket(id: number, changes: { status: Status }) {
    return request<Ticket>(`/tickets/${id}`, { method: "PATCH", body: JSON.stringify(changes) });
  },
  stats() {
    return request<Stats>("/tickets/stats");
  },
};
