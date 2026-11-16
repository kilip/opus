import { Link } from "react-router";

import { buttonClass } from "~/shared/components/Button";

export function EmptyTasks() {
	return (
		<div className="flex flex-1 items-center justify-center p-8">
			<div className="flex max-w-sm flex-col items-center gap-3 rounded-lg border border-line bg-surface p-8 text-center">
				<h2 className="text-lg font-semibold">No tasks yet</h2>
				<p className="text-sm text-muted">
					Create your first task to get started.
				</p>
				<Link to="/tasks/new" className={buttonClass("primary")}>
					Create task
				</Link>
			</div>
		</div>
	);
}
