export const DEFAULT_REPO_PROVIDER = "github";
export const DEFAULT_BRANCH = "main";

export const TRACKER_STATUS_ORDER = [
  "backlog",
  "todo",
  "in-progress",
  "review",
  "blocked",
] as const;

export type TrackerStatusKey = (typeof TRACKER_STATUS_ORDER)[number];

export const DEFAULT_STATUS_LABELS: Record<TrackerStatusKey, string> = {
  backlog: "status:backlog",
  todo: "status:todo",
  "in-progress": "status:in-progress",
  review: "status:review",
  blocked: "status:blocked",
};

export type CreateProjectDraft = {
  slug: string;
  name: string;
  repoRef: string;
  defaultBranch: string;
};

export const DEFAULT_CREATE_PROJECT_DRAFT: CreateProjectDraft = {
  slug: "",
  name: "",
  repoRef: "",
  defaultBranch: DEFAULT_BRANCH,
};

export function formatRepoRef(
  _provider: string | undefined,
  repoRef: string | undefined,
): string {
  if (!repoRef) return "";
  return repoRef;
}
