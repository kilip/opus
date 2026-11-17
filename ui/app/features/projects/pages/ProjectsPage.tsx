import { type ReactNode, useState } from "react";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Icon } from "~/shared/components/Icon";
import { Tag } from "~/shared/components/Tag";
import { useProjects } from "../api/queries";
import type { Project } from "../api/types";
import { CreateProjectModal } from "../components/CreateProjectModal";
import { EditProjectModal } from "../components/EditProjectModal";

export const meta = () => [
	{ title: "Projects · Opus" },
	{ name: "description", content: "Manage projects" },
];

export type ProjectsPageProps = {
	initialEditingProject?: Project | null;
	initialIsCreateOpen?: boolean;
};

export default function ProjectsPage({
	initialEditingProject = null,
	initialIsCreateOpen = false,
}: ProjectsPageProps = {}) {
	const projectsQuery = useProjects();
	const [isCreateOpen, setIsCreateOpen] = useState(initialIsCreateOpen);
	const [editingProject, setEditingProject] = useState<Project | null>(
		initialEditingProject,
	);

	const projects = projectsQuery.data ?? [];
	const total = projects.length;

	let content: ReactNode;
	if (projectsQuery.isPending) {
		content = <p className="p-8 text-center text-muted">Loading projects…</p>;
	} else if (projectsQuery.isError) {
		content = (
			<div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
				<p className="text-lg font-semibold">Could not load projects</p>
				<p className="text-muted">{projectsQuery.error?.message}</p>
				<Button variant="primary" onClick={() => projectsQuery.refetch()}>
					Retry
				</Button>
			</div>
		);
	} else if (total === 0) {
		content = (
			<div className="rounded-lg border border-line bg-surface p-8 text-center text-muted">
				<p className="font-semibold text-fg">No projects yet</p>
				<p className="mt-1 text-sm">
					Create a project to start managing tasks and node paths.
				</p>
				<div className="mt-4">
					<Button variant="primary" onClick={() => setIsCreateOpen(true)}>
						Create Project
					</Button>
				</div>
			</div>
		);
	} else {
		content = (
			<div className="overflow-hidden rounded-lg border border-line bg-surface">
				<table className="w-full text-left text-sm">
					<thead className="border-b border-line bg-surface-2 text-xs font-semibold uppercase tracking-wider text-muted">
						<tr>
							<th className="px-4 py-3">Project</th>
							<th className="px-4 py-3">Repository</th>
							<th className="px-4 py-3">Default Branch</th>
							<th className="px-4 py-3 text-right">Actions</th>
						</tr>
					</thead>
					<tbody className="divide-y divide-line">
						{projects.map((project) => (
							<tr
								key={project.id ?? project.slug}
								className="hover:bg-surface-2/50"
							>
								<td className="px-4 py-3 font-medium text-fg">
									<div className="flex flex-col">
										<span>{project.name || "Unnamed Project"}</span>
										<span className="font-mono text-xs text-muted">
											{project.slug}
										</span>
									</div>
								</td>
								<td className="px-4 py-3">
									<div className="flex items-center gap-1.5 font-mono text-xs text-fg">
										<Icon name="github" size={14} className="text-muted" />
										<span>{project.repo_ref || "—"}</span>
									</div>
								</td>
								<td className="px-4 py-3">
									<Tag tone="neutral">{project.default_branch || "main"}</Tag>
								</td>
								<td className="px-4 py-3 text-right">
									<Button
										variant="secondary"
										onClick={() => setEditingProject(project)}
									>
										Edit
									</Button>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			</div>
		);
	}

	return (
		<>
			<ControlPanel
				breadcrumbs={[{ label: `Projects (${total})` }]}
				actions={
					<Button variant="primary" onClick={() => setIsCreateOpen(true)}>
						Create Project
					</Button>
				}
			/>

			<div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 p-6">
				{content}
			</div>

			<CreateProjectModal
				isOpen={isCreateOpen}
				onClose={() => setIsCreateOpen(false)}
			/>

			{editingProject && (
				<EditProjectModal
					project={editingProject}
					isOpen={Boolean(editingProject)}
					onClose={() => setEditingProject(null)}
				/>
			)}
		</>
	);
}
