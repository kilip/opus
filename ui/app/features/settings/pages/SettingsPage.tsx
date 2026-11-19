import { type FormEvent, useState } from "react";

import { Button } from "~/shared/components/Button";
import { useDeleteGitHubToken, useSetGitHubToken } from "../api/actions";
import { useGitHubTokenStatus } from "../api/queries";

export const meta = () => [
  { title: "Settings · Opus" },
  {
    name: "description",
    content: "Configure encrypted runtime credentials and integrations",
  },
];

export default function SettingsPage() {
  const [tokenInput, setTokenInput] = useState("");
  const statusQuery = useGitHubTokenStatus();
  const setToken = useSetGitHubToken();
  const deleteToken = useDeleteGitHubToken();

  const status = statusQuery.data;
  const error =
    setToken.error?.message ??
    deleteToken.error?.message ??
    statusQuery.error?.message ??
    null;

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const trimmed = tokenInput.trim();
    if (!trimmed) {
      return;
    }
    setToken.mutate(
      { token: trimmed },
      {
        onSuccess: () => {
          setTokenInput("");
        },
      },
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold text-fg">Settings</h1>
        <p className="text-sm text-muted">
          Manage runtime credentials stored encrypted at rest.
        </p>
      </div>

      {error && (
        <div
          role="alert"
          className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
        >
          {error}
        </div>
      )}

      <section className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-1">
            <h2 className="text-base font-medium text-fg">GitHub Token</h2>
            <p className="text-sm text-muted">
              Used by the tracker sync and{" "}
              <code className="font-mono text-xs">gh</code> CLI. Encrypted with{" "}
              <code className="font-mono text-xs">AUTH_SECRET</code>; rotating{" "}
              <code className="font-mono text-xs">AUTH_SECRET</code> requires
              re-entering stored tokens.
            </p>
          </div>
          {status?.configured && (
            <span className="shrink-0 rounded-full border border-line px-2.5 py-0.5 text-xs font-medium text-fg">
              ••••{status.last4}
            </span>
          )}
        </div>

        {status?.envOverride && (
          <div
            role="status"
            className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-200"
          >
            The <code className="font-mono text-xs">GH_TOKEN</code> environment
            variable is set and overrides the stored token.
          </div>
        )}

        {status?.needsReentry && (
          <div
            role="alert"
            className="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            Stored token could not be decrypted because{" "}
            <code className="font-mono text-xs">AUTH_SECRET</code> changed.
            Please re-enter your GitHub token.
          </div>
        )}

        {statusQuery.isPending ? (
          <div className="text-sm text-muted">Loading token status…</div>
        ) : (
          <div className="text-xs text-muted">
            {status?.configured && status.updatedAt
              ? `Configured (••••${status.last4}) · Updated ${new Date(status.updatedAt).toLocaleString()}`
              : "No stored GitHub token configured."}
          </div>
        )}

        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-3 sm:flex-row"
        >
          <input
            type="password"
            name="githubToken"
            aria-label="GitHub token"
            placeholder="ghp_... or github_pat_..."
            autoComplete="off"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            className="flex-1 rounded-md border border-line bg-bg px-3 py-2 text-sm text-fg placeholder:text-muted focus:border-accent focus:outline-none"
          />
          <div className="flex items-center gap-2">
            <Button
              type="submit"
              disabled={setToken.isPending || !tokenInput.trim()}
            >
              {setToken.isPending ? "Saving…" : "Save"}
            </Button>
            {(status?.configured || status?.needsReentry) && (
              <Button
                type="button"
                variant="secondary"
                disabled={deleteToken.isPending}
                onClick={() => deleteToken.mutate()}
              >
                {deleteToken.isPending ? "Removing…" : "Remove"}
              </Button>
            )}
          </div>
        </form>
      </section>
    </div>
  );
}
