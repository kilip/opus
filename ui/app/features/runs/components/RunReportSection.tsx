import { Tag } from "~/shared/components/Tag";
import { parseRunReport } from "../model";

export function RunReportSection({ report }: { report?: unknown }) {
  const parsed = parseRunReport(report);

  if (!parsed) {
    return (
      <div className="rounded-lg border border-line bg-surface p-4 text-sm text-muted">
        No report submitted for this run yet.
      </div>
    );
  }

  const isHttpUrl = (value: string) =>
    value.startsWith("https://") || value.startsWith("http://");

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-4 text-sm">
      {parsed.summary && (
        <div>
          <span className="text-xs text-muted">Summary</span>
          <p className="mt-1 whitespace-pre-wrap text-fg">{parsed.summary}</p>
        </div>
      )}

      {(parsed.pr_url || parsed.commit_sha) && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <span className="text-xs text-muted">PR URL</span>
            {parsed.pr_url ? (
              isHttpUrl(parsed.pr_url) ? (
                <p className="mt-1 break-all font-mono text-xs">
                  <a
                    href={parsed.pr_url}
                    target="_blank"
                    rel="noreferrer"
                    className="text-primary hover:underline"
                  >
                    {parsed.pr_url}
                  </a>
                </p>
              ) : (
                <p className="mt-1 break-all font-mono text-xs text-fg">
                  {parsed.pr_url}
                </p>
              )
            ) : (
              <p className="mt-1 font-mono text-xs text-muted">—</p>
            )}
          </div>
          <div>
            <span className="text-xs text-muted">Commit SHA</span>
            <p className="mt-1 font-mono text-xs text-fg">
              {parsed.commit_sha ?? "—"}
            </p>
          </div>
        </div>
      )}

      {parsed.assumptions && parsed.assumptions.length > 0 && (
        <div>
          <span className="text-xs text-muted">Assumptions</span>
          <ul className="mt-1 list-disc space-y-1 pl-5 text-fg">
            {parsed.assumptions.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
      )}

      {parsed.acceptance_criteria && parsed.acceptance_criteria.length > 0 && (
        <div>
          <span className="text-xs text-muted">Acceptance Criteria</span>
          <ul className="mt-2 space-y-2">
            {parsed.acceptance_criteria.map((ac) => (
              <li
                key={ac.id ?? ac.criterion ?? "ac"}
                className="rounded border border-line bg-surface-2 px-3 py-2"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-medium text-fg">
                    {`${ac.id ? `${ac.id}: ` : ""}${ac.criterion ?? "Criterion"}`}
                  </span>
                  <Tag tone={ac.met ? "success" : "danger"}>
                    {ac.met ? "Met" : "Unmet"}
                  </Tag>
                </div>
                {ac.evidence && (
                  <p className="mt-1 font-mono text-xs text-muted">
                    {ac.evidence}
                  </p>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
