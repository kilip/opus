import { useSearchParams } from "react-router";

import { Button, buttonClass } from "~/shared/components/Button";
import { Icon, type IconName, isIconName } from "~/shared/components/Icon";
import { useDisconnectAccount } from "../api/actions";
import { resolveAuthError } from "../api/errors";
import { useAccounts, useProviders } from "../api/queries";

export const meta = () => [
  { title: "Connected Accounts · Opus" },
  {
    name: "description",
    content: "Manage external OAuth accounts connected to your profile",
  },
];

export default function ConnectedAccountsPage() {
  const [searchParams] = useSearchParams();
  const accountsQuery = useAccounts();
  const providersQuery = useProviders();
  const disconnect = useDisconnectAccount();

  const urlError = resolveAuthError(searchParams.get("error"));
  const mutationError = disconnect.error ? disconnect.error.message : null;
  const error = urlError || mutationError;

  const accounts = accountsQuery.data ?? [];
  const providers = providersQuery.data ?? [];

  const linkedProviders = new Set(
    accounts.map((a) => (a.provider ?? "").toLowerCase()),
  );

  const unlinkedProviders = providers.filter(
    (p) => p.name && !linkedProviders.has(p.name.toLowerCase()),
  );

  const providerLabels = new Map(
    providers.map((p) => [p.name?.toLowerCase(), p.label || p.name]),
  );

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold text-fg">Connected Accounts</h1>
        <p className="text-sm text-muted">
          Manage external authentication providers linked to your account.
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

      <section className="flex flex-col gap-3">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted">
          Linked Accounts
        </h2>
        {accountsQuery.isPending ? (
          <div className="p-4 text-center text-sm text-muted">
            Loading accounts…
          </div>
        ) : accounts.length === 0 ? (
          <div className="rounded-lg border border-line bg-surface p-6 text-center text-sm text-muted">
            No connected accounts yet.
          </div>
        ) : (
          <div className="divide-y divide-line rounded-lg border border-line bg-surface">
            {accounts.map((acc) => {
              const provName = (acc.provider ?? "").toLowerCase();
              const label = providerLabels.get(provName) || acc.provider;
              const iconName = isIconName(provName)
                ? (provName as IconName)
                : undefined;
              const isDisconnecting =
                disconnect.isPending && disconnect.variables === acc.id;

              return (
                <div
                  key={acc.id ?? acc.provider}
                  className="flex items-center justify-between gap-4 p-4"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    {iconName && (
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-surface-2 text-fg">
                        <Icon name={iconName} size={18} />
                      </div>
                    )}
                    <div className="flex flex-col min-w-0">
                      <span className="font-medium text-fg capitalize truncate">
                        {label}
                      </span>
                      {acc.email && (
                        <span className="text-xs text-muted truncate">
                          {acc.email}
                        </span>
                      )}
                    </div>
                  </div>
                  <Button
                    variant="secondary"
                    disabled={isDisconnecting}
                    onClick={() => {
                      if (acc.id) {
                        disconnect.mutate(acc.id);
                      }
                    }}
                    aria-label={`Disconnect ${label}`}
                  >
                    {isDisconnecting ? "Disconnecting…" : "Disconnect"}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </section>

      {providers.length > 0 && (
        <section className="flex flex-col gap-3">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted">
            Available Providers
          </h2>
          {unlinkedProviders.length === 0 ? (
            <div className="rounded-lg border border-line bg-surface p-6 text-center text-sm text-muted">
              All available providers are connected.
            </div>
          ) : (
            <div className="divide-y divide-line rounded-lg border border-line bg-surface">
              {unlinkedProviders.map((provider) => {
                const provName = (provider.name ?? "").toLowerCase();
                const iconName = isIconName(provName)
                  ? (provName as IconName)
                  : undefined;
                const label = provider.label || provider.name;

                return (
                  <div
                    key={provider.name}
                    className="flex items-center justify-between gap-4 p-4"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      {iconName && (
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-surface-2 text-fg">
                          <Icon name={iconName} size={18} />
                        </div>
                      )}
                      <span className="font-medium text-fg truncate">
                        {label}
                      </span>
                    </div>
                    <a
                      href={`/api/auth/oauth/${provider.name}?link=1`}
                      className={buttonClass("secondary", "gap-1.5")}
                      aria-label={`Connect ${label}`}
                    >
                      Connect
                    </a>
                  </div>
                );
              })}
            </div>
          )}
        </section>
      )}
    </div>
  );
}
