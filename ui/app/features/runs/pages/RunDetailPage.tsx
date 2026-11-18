import { Link, useParams } from "react-router";

import { Button } from "~/shared/components/Button";
import { ControlPanel } from "~/shared/components/ControlPanel";
import { Tag } from "~/shared/components/Tag";
import { formatAbsoluteTime } from "~/shared/util/dates";
import { useCancelRun } from "../api/actions";
import { useRun } from "../api/queries";
import { RunReportSection } from "../components/RunReportSection";
import { RunStateBadge } from "../components/RunStateBadge";
import { SessionLogStream } from "../components/SessionLogStream";
import { buildStateTimeline, isActiveRunState } from "../model";

export const meta = () => [
  { title: "Run Detail · Opus" },
  { name: "description", content: "Inspect run state timeline and logs" },
];

export default function RunDetailPage({
  initialRunId,
}: {
  initialRunId?: string;
} = {}) {
  const { id = initialRunId ?? "" } = useParams();
  const runQuery = useRun(id);
  const cancelRun = useCancelRun();

  if (runQuery.isPending) {
    return <p className="p-8 text-center text-muted">Loading run…</p>;
  }
  if (runQuery.isError || !runQuery.data) {
    return (
      <div className="mx-auto flex max-w-2xl flex-col items-center gap-3 p-8 text-center">
        <p className="text-lg font-semibold">Could not load run</p>
        <p className="text-muted">{runQuery.error?.message}</p>
        <Link to="/runs" className="text-sm text-primary hover:underline">
          Back to runs
        </Link>
      </div>
    );
  }

  const run = runQuery.data;
  const runId = run.id ?? id;
  const runState = run.state ?? "queued";
  const timeline = buildStateTimeline(run);
  const canCancel = isActiveRunState(runState);

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-6">
      <ControlPanel
        breadcrumbs={[
          { label: "Runs", to: "/runs" },
          { label: runId.slice(0, 8) },
        ]}
        right={
          canCancel ? (
            <Button
              variant="secondary"
              disabled={cancelRun.isPending}
              onClick={() => cancelRun.mutate({ id: runId })}
            >
              {cancelRun.isPending ? "Cancelling…" : "Cancel Run"}
            </Button>
          ) : undefined
        }
      />

      {cancelRun.error && (
        <div className="rounded-lg border border-danger bg-danger-soft px-4 py-2 text-sm text-danger-ink">
          {cancelRun.error.message}
        </div>
      )}

      <div className="rounded-lg border border-line bg-surface p-4">
        <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-muted">
          State Timeline
        </h2>
        <div className="flex flex-wrap items-center gap-3">
          {timeline.map((step, i) => (
            <div key={step.state} className="flex items-center gap-3">
              <Tag
                tone={
                  step.status === "done"
                    ? "success"
                    : step.status === "current"
                      ? "primary"
                      : step.status === "failed"
                        ? "danger"
                        : "neutral"
                }
              >
                {step.label}
              </Tag>
              {i < timeline.length - 1 && (
                <span className="text-xs text-muted">→</span>
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 rounded-lg border border-line bg-surface p-4 text-sm sm:grid-cols-2">
        <div>
          <span className="text-xs text-muted">State</span>
          <div className="mt-1">
            <RunStateBadge state={runState} />
          </div>
        </div>
        <div>
          <span className="text-xs text-muted">Mode</span>
          <p className="mt-1 font-mono">{run.mode ?? "vibe"}</p>
        </div>
        <div>
          <span className="text-xs text-muted">Task ID</span>
          <p className="mt-1 font-mono text-xs">{run.taskId ?? "—"}</p>
        </div>
        <div>
          <span className="text-xs text-muted">Project ID</span>
          <p className="mt-1 font-mono text-xs">{run.projectId ?? "—"}</p>
        </div>
        <div>
          <span className="text-xs text-muted">Node ID</span>
          <p className="mt-1 font-mono text-xs">{run.nodeId ?? "—"}</p>
        </div>
        <div>
          <span className="text-xs text-muted">Branch</span>
          <p className="mt-1 font-mono text-xs">{run.branch || "—"}</p>
        </div>
        <div>
          <span className="text-xs text-muted">Created</span>
          <p className="mt-1 text-xs">
            {run.createdAt ? formatAbsoluteTime(run.createdAt) : "—"}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted">Started / Ended</span>
          <p className="mt-1 text-xs">
            {run.startedAt ? formatAbsoluteTime(run.startedAt) : "—"}
            {" / "}
            {run.endedAt ? formatAbsoluteTime(run.endedAt) : "—"}
          </p>
        </div>
        {run.failureReason && (
          <div className="sm:col-span-2">
            <span className="text-xs text-muted">Failure Reason</span>
            <p className="mt-1 text-danger-ink">{run.failureReason}</p>
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted">
          Agent Report
        </h2>
        <RunReportSection report={run.report} />
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted">
          Session Log
        </h2>
        <SessionLogStream sessionId={run.sessionId} />
      </div>
    </div>
  );
}
