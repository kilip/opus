export const PROJECT_ERRORS: Record<string, string> = {
  PROJECT_NOT_FOUND: "The requested project was not found.",
  PROJECT_INVALID_SLUG:
    "Project slug must be 1-63 lowercase alphanumeric characters or hyphens.",
  PROJECT_DUPLICATE_SLUG: "A project with this slug already exists.",
  PROJECT_INVALID_REPO: "Repository must be in owner/name format.",
  PROJECT_PATCH_FORBIDDEN:
    "Slug and repository cannot be changed after creation.",
  PROJECT_HAS_TASKS:
    "Project cannot be deleted because it still has associated tasks.",
  PROJECT_INVALID_PATH: "Node path must be an absolute path starting with '/'.",
  NODE_NOT_FOUND: "Node not found.",
  TRACKER_INVALID_LABEL_CONFIG:
    "Status labels must be non-empty, unique, and at most 50 characters.",
};
