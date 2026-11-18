import { Link } from "react-router";

import { ControlPanel } from "~/shared/components/ControlPanel";
import { Tag } from "~/shared/components/Tag";
import { formatRelativeTime } from "~/shared/util/dates";
import {
	useDashboardNodes,
	useDashboardRuns,
	useDashboardTasks,
} from "../api/queries";
import { summarizeDashboard, TASK_STATUS_ORDER } from "../model";

export const meta = () => [
	{ title: "Dashboard · Opus" },
	{ name: "description", content: "Opus cluster and execution overview" },
];

export default function DashboardPage() {
	const nodesQuery = useDashboardNodes();
	const tasksQuery = useDashboardTasks();
	const runsQuery = useDashboardRuns();

	const summary = summarizeDashboard(
		nodesQuery.data ?? [],
		tasksQuery.data ?? [],
		runsQuery.data ?? [],
	);

	const isLoading =
		nodesQuery.isPending || tasksQuery.isPending || runsQuery.isPending;

	return (
		<div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6">
			<ControlPanel breadcrumbs={[{ label: "Dashboard" }]} />

			{isLoading && (
				<p className="text-sm text-muted">Loading cluster overview…</p>
			)}

			<div className="grid grid-cols-1 gap-4 md:grid-cols-2">
				<section className="rounded-lg border border-line bg-surface p-4">
					<div className="mb-3 flex items-center justify-between">
						<h2 className="text-sm font-semibold">Nodes Online</h2>
						<Link to="/nodes" className="text-xs text-primary hover:underline">
							{summary.onlineNodes.length} / {summary.totalNodes} online
						</Link>
					</div>
					{summary.onlineNodes.length === 0 ? (
						<p className="text-sm text-muted">No nodes online.</p>
					) : (
						<ul className="space-y-2 text-sm">
							{summary.onlineNodes.map((n) => (
								<li
									key={n.id}
									className="flex items-center justify-between rounded bg-surface-2 px-3 py-1.5"
								>
									<span className="font-medium">{n.name}</span>
									<Tag tone="success">{n.role}</Tag>
								</li>
							))}
						</ul>
					)}
				</section>

				<section className="rounded-lg border border-line bg-surface p-4">
					<div className="mb-3 flex items-center justify-between">
						<h2 className="text-sm font-semibold">Tasks by Status</h2>
						<Link to="/tasks" className="text-xs text-primary hover:underline">
							View tasks
						</Link>
					</div>
					<div className="grid grid-cols-3 gap-2">
						{TASK_STATUS_ORDER.map((st) => (
							<div
								key={st}
								className="rounded bg-surface-2 px-3 py-2 text-center"
							>
								<p className="text-lg font-semibold">
									{summary.taskCounts[st]}
								</p>
								<p className="text-xs text-muted">{st}</p>
							</div>
						))}
					</div>
				</section>

				<section className="rounded-lg border border-line bg-surface p-4">
					<div className="mb-3 flex items-center justify-between">
						<h2 className="text-sm font-semibold">Active Runs</h2>
						<Link to="/runs" className="text-xs text-primary hover:underline">
							{summary.activeRuns.length} active
						</Link>
					</div>
					{summary.activeRuns.length === 0 ? (
						<p className="text-sm text-muted">No active runs.</p>
					) : (
						<ul className="space-y-2 text-sm">
							{summary.activeRuns.map((r) => (
								<li
									key={r.id}
									className="flex items-center justify-between rounded bg-surface-2 px-3 py-1.5"
								>
									<Link
										to={`/runs/${r.id}`}
										className="font-mono text-xs text-primary hover:underline"
									>
										{(r.id ?? "").slice(0, 8)}
									</Link>
									<Tag tone="primary">{r.state ?? "queued"}</Tag>
								</li>
							))}
						</ul>
					)}
				</section>

				<section className="rounded-lg border border-line bg-surface p-4">
					<div className="mb-3 flex items-center justify-between">
						<h2 className="text-sm font-semibold">Recent Run Outcomes</h2>
						<Link to="/runs" className="text-xs text-primary hover:underline">
							All runs
						</Link>
					</div>
					{summary.recentOutcomes.length === 0 ? (
						<p className="text-sm text-muted">No completed runs yet.</p>
					) : (
						<ul className="space-y-2 text-sm">
							{summary.recentOutcomes.map((r) => (
								<li
									key={r.id}
									className="flex items-center justify-between rounded bg-surface-2 px-3 py-1.5"
								>
									<Link
										to={`/runs/${r.id}`}
										className="font-mono text-xs text-primary hover:underline"
									>
										{(r.id ?? "").slice(0, 8)}
									</Link>
									<div className="flex items-center gap-2">
										<span className="text-xs text-muted">
											{r.endedAt ? formatRelativeTime(r.endedAt) : "—"}
										</span>
										<Tag tone={r.state === "merged" ? "success" : "danger"}>
											{r.state ?? "failed"}
										</Tag>
									</div>
								</li>
							))}
						</ul>
					)}
				</section>
			</div>
		</div>
	);
}
