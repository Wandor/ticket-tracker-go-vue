export const STATUSES = ["open", "in_progress", "closed"] as const;
export const PRIORITIES = ["low", "medium", "high"] as const;

export type Status = (typeof STATUSES)[number];
export type Priority = (typeof PRIORITIES)[number];

export const STATUS_LABELS: Record<Status, string> = {
  open: "Open",
  in_progress: "In progress",
  closed: "Closed",
};

export interface Ticket {
  id: number;
  title: string;
  description: string | null;
  status: Status;
  priority: Priority;
  created_at: string;
}

export interface Page<T> {
  data: T[];
  pagination: { page: number; limit: number; total: number; total_pages: number };
}

export type Stats = Record<Status, number>;

export interface NewTicket {
  title: string;
  description: string;
  priority: Priority;
}
