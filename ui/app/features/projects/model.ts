export const DEFAULT_REPO_PROVIDER = "github";
export const DEFAULT_BRANCH = "main";

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
