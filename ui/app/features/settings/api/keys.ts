export const settingsKeys = {
  all: ["settings"] as const,
  githubToken: () => [...settingsKeys.all, "github-token"] as const,
};
