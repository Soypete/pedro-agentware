import { Job, JobStatus } from "./job.js";

export interface JobManager {
  create(description: string): string;
  start(jobId: string): void;
  complete(jobId: string, result: string): void;
  fail(jobId: string, error: string): void;
  cancel(jobId: string): void;
  get(jobId: string): Job;
  list(status?: JobStatus): Job[];
}

export class InMemoryJobManager implements JobManager {
  private jobs: Map<string, Job> = new Map();
  private counter = 0;
  private lock = new Map<string, true>();

  create(description: string): string {
    this.counter++;
    const jobId = `${Date.now().toString().slice(0, 14)}-${String(this.counter).padStart(4, "0")}`;
    const now = new Date();
    const job: Job = {
      id: jobId,
      status: JobStatus.PENDING,
      description,
      created_at: now,
      updated_at: now,
      result: "",
      error: "",
    };
    this.jobs.set(jobId, job);
    return jobId;
  }

  start(jobId: string): void {
    const job = this.jobs.get(jobId);
    if (!job) throw new Error(`job not found: ${jobId}`);
    job.status = JobStatus.RUNNING;
    job.updated_at = new Date();
  }

  complete(jobId: string, result: string): void {
    const job = this.jobs.get(jobId);
    if (!job) throw new Error(`job not found: ${jobId}`);
    job.status = JobStatus.COMPLETE;
    job.result = result;
    job.updated_at = new Date();
  }

  fail(jobId: string, error: string): void {
    const job = this.jobs.get(jobId);
    if (!job) throw new Error(`job not found: ${jobId}`);
    job.status = JobStatus.FAILED;
    job.error = error;
    job.updated_at = new Date();
  }

  cancel(jobId: string): void {
    const job = this.jobs.get(jobId);
    if (!job) throw new Error(`job not found: ${jobId}`);
    job.status = JobStatus.CANCELED;
    job.updated_at = new Date();
  }

  get(jobId: string): Job {
    const job = this.jobs.get(jobId);
    if (!job) throw new Error(`job not found: ${jobId}`);
    return job;
  }

  list(status?: JobStatus): Job[] {
    if (!status) return Array.from(this.jobs.values());
    return Array.from(this.jobs.values()).filter((j) => j.status === status);
  }
}