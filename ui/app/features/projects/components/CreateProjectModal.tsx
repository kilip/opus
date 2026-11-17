import { type FormEvent, useState } from "react";

import { Button } from "~/shared/components/Button";
import { Icon } from "~/shared/components/Icon";
import { useCreateProject } from "../api/actions";
import type { Project } from "../api/types";
import { DEFAULT_BRANCH, DEFAULT_REPO_PROVIDER } from "../model";

export type CreateProjectModalProps = {
	isOpen: boolean;
	onClose: () => void;
	onSuccess?: (project: Project) => void;
};

export function CreateProjectModal({
	isOpen,
	onClose,
	onSuccess,
}: CreateProjectModalProps) {
	const createProject = useCreateProject();

	const [slug, setSlug] = useState("");
	const [name, setName] = useState("");
	const [repoRef, setRepoRef] = useState("");
	const [defaultBranch, setDefaultBranch] = useState(DEFAULT_BRANCH);

	if (!isOpen) {
		return null;
	}

	const errorMessage = createProject.error?.message ?? null;

	function handleSubmit(e: FormEvent) {
		e.preventDefault();
		createProject.mutate(
			{
				slug: slug.trim(),
				name: name.trim(),
				repo_provider: DEFAULT_REPO_PROVIDER,
				repo_ref: repoRef.trim(),
				default_branch: defaultBranch.trim() || DEFAULT_BRANCH,
			},
			{
				onSuccess: (project) => {
					setSlug("");
					setName("");
					setRepoRef("");
					setDefaultBranch(DEFAULT_BRANCH);
					onSuccess?.(project);
					onClose();
				},
			},
		);
	}

	return (
		<div
			role="dialog"
			aria-modal="true"
			aria-labelledby="create-project-title"
			className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		>
			<div className="w-full max-w-lg rounded-lg border border-line bg-surface p-6 shadow-pop">
				<div className="flex items-center justify-between border-b border-line pb-3">
					<h2
						id="create-project-title"
						className="text-lg font-semibold text-fg"
					>
						Create Project
					</h2>
					<button
						type="button"
						onClick={onClose}
						className="rounded p-1 text-muted hover:bg-surface-2 hover:text-fg"
						aria-label="Close"
					>
						<Icon name="x" size={16} />
					</button>
				</div>

				<form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-4">
					{errorMessage && (
						<div
							role="alert"
							className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
						>
							{errorMessage}
						</div>
					)}

					<div>
						<label
							htmlFor="project-slug"
							className="block text-xs font-semibold uppercase tracking-wider text-muted"
						>
							Slug
						</label>
						<input
							id="project-slug"
							name="slug"
							type="text"
							required
							pattern="[a-z0-9][a-z0-9-]{0,62}"
							placeholder="my-project"
							value={slug}
							onChange={(e) => setSlug(e.target.value)}
							className="field mt-1"
						/>
						<p className="mt-1 text-xs text-muted">
							Identifier for the project (lowercase alphanumeric and hyphens).
						</p>
					</div>

					<div>
						<label
							htmlFor="project-name"
							className="block text-xs font-semibold uppercase tracking-wider text-muted"
						>
							Name
						</label>
						<input
							id="project-name"
							name="name"
							type="text"
							required
							placeholder="My Project"
							value={name}
							onChange={(e) => setName(e.target.value)}
							className="field mt-1"
						/>
					</div>

					<div>
						<label
							htmlFor="project-repo"
							className="block text-xs font-semibold uppercase tracking-wider text-muted"
						>
							Repository (GitHub)
						</label>
						<div className="mt-1 flex items-center gap-2">
							<span className="flex items-center gap-1 text-xs font-medium text-muted">
								<Icon name="github" size={14} />
								github.com/
							</span>
							<input
								id="project-repo"
								name="repo_ref"
								type="text"
								required
								placeholder="owner/repo"
								value={repoRef}
								onChange={(e) => setRepoRef(e.target.value)}
								className="field flex-1"
							/>
						</div>
					</div>

					<div>
						<label
							htmlFor="project-branch"
							className="block text-xs font-semibold uppercase tracking-wider text-muted"
						>
							Default Branch
						</label>
						<input
							id="project-branch"
							name="default_branch"
							type="text"
							placeholder="main"
							value={defaultBranch}
							onChange={(e) => setDefaultBranch(e.target.value)}
							className="field mt-1"
						/>
					</div>

					<div className="mt-4 flex items-center justify-end gap-2 border-t border-line pt-4">
						<Button
							variant="secondary"
							onClick={onClose}
							disabled={createProject.isPending}
						>
							Cancel
						</Button>
						<Button
							type="submit"
							variant="primary"
							disabled={createProject.isPending}
						>
							{createProject.isPending ? "Creating…" : "Create Project"}
						</Button>
					</div>
				</form>
			</div>
		</div>
	);
}
