import { type ReactNode, useState } from "react";
import { Link } from "react-router";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Tag } from "~/shared/components/Tag";
import { formatRelativeTime } from "~/shared/util/dates";
import { useRuns } from "../api/queries";
import type { RunState } from "../api/types";
import { RunStateBadge } from "../components/RunStateBadge";
import { RUN_STATES } from "../model";

export const meta = () => [
	{ title: "Runs · Opus" },
	{ name: "description", content: "Inspect automated agent runs" },
];

export default function RunsPage() {
	const [stateFilter, setStateFilter] = useState<RunState | undefined>(
		undefined,
	);
	const runsQuery = useRuns(stateFilter);
	const runs = runsQuery.data ?? [];

	let content: ReactNode;
	if (runsQuery.isPending) {
		content = <p className="p-8 text-center text-muted">Loading runs…</p>;
	} else if (runsQuery.isError) {
		content = (
			<div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
				<p className="text-lg font-semibold">Could not load runs</p>
				<p className="text-muted">{runsQuery.error?.message}</p>
				<Button variant="primary" onClick={() => runsQuery.refetch()}>
					Retry
				</Button>
			</div>
		);
	} else if (runs.length === 0) {
		content = (
			<div className="rounded-lg border border-line bg-surface p-8 text-center text-muted">
				<p className="font-semibold text-fg">No runs found</p>
				<p className="mt-1 text-sm">
					Runs appear here when the reconciler dispatches eligible tasks.
				</p>
			</div>
		);
	} else {
		content = (
			<div className="overflow-x-auto rounded-lg border border-line bg-surface">
				<table className="w-full text-left text-sm">
					<thead className="border-b border-line text-xs uppercase text-muted">
						<tr>
							<th className="px-4 py-3">Run</th>
							<th className="px-4 py-3">State</th>
							<th className="px-4 py-3">Mode</th>
							<th className="px-4 py-3">Task</th>
							<th className="px-4 py-3">Project</th>
							<th className="px-4 py-3">Node</th>
							<th className="px-4 py-3">Created</th>
						</tr>
					</thead>
					<tbody className="divide-y divide-line">
						{runs.map((r) => {
							const runId = r.id ?? "";
							return (
								<tr key={runId} className="hover:bg-surface-2">
									<td className="px-4 py-3 font-mono text-xs">
										<Link
											to={`/runs/${runId}`}
											className="text-primary hover:underline"
										>
											{runId.slice(0, 8)}
										</Link>
									</td>
									<td className="px-4 py-3">
										<RunStateBadge state={r.state} />
									</td>
									<td className="px-4 py-3">
										<Tag tone="neutral">{r.mode ?? "vibe"}</Tag>
									</td>
									<td className="px-4 py-3 font-mono text-xs text-muted">
										{(r.taskId ?? "").slice(0, 8)}
									</td>
									<td className="px-4 py-3 font-mono text-xs text-muted">
										{(r.projectId ?? "").slice(0, 8)}
									</td>
									<td className="px-4 py-3 font-mono text-xs text-muted">
										{r.nodeId ? r.nodeId.slice(0, 8) : "—"}
									</td>
									<td className="px-4 py-3 text-xs text-muted">
										{r.createdAt ? formatRelativeTime(r.createdAt) : "—"}
									</td>
								</tr>
							);
						})}
					</tbody>
				</table>
			</div>
		);
	}

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6">
			<ControlPanel
				breadcrumbs={[{ label: `Runs (${runs.length})` }]}
				right={
					<div className="flex flex-wrap items-center gap-1.5">
						<Button
							variant={stateFilter === undefined ? "primary" : "ghost"}
							onClick={() => setStateFilter(undefined)}
						>
							All
						</Button>
						{RUN_STATES.map((st) => (
							<Button
								key={st}
								variant={stateFilter === st ? "primary" : "ghost"}
								onClick={() => setStateFilter(st)}
							>
								{st}
							</Button>
						))}
					</div>
				}
			/>
			{content}
		</div>
	);
}
