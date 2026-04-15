export enum JobStatus {
  PENDING = "pending",
  RUNNING = "running",
  COMPLETE = "complete",
  FAILED = "failed",
  CANCELED = "canceled",
}

export interface Job {
  id: string;
  status: JobStatus;
  description: string;
  created_at: Date;
  updated_at: Date;
  result: string;
  error: string;
}

export function createJob(id: string, description: string): Job {
  const now = new Date();
  return {
    id,
    status: JobStatus.PENDING,
    description,
    created_at: now,
    updated_at: now,
    result: "",
    error: "",
  };
}