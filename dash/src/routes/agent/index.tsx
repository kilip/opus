import { createFileRoute } from '@tanstack/react-router';

/**
 * Route definition for the agent feature dashboard.
 */
export const Route = createFileRoute('/agent/')({
  component: AgentPage,
});

/**
 * AgentPage component displaying the list of agents and administrative overview.
 */
export function AgentPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-extrabold tracking-tight text-white">Agents</h2>
          <p className="text-sm text-slate-400 mt-1">
            Manage your autonomous AI agents and supervise their active pipelines.
          </p>
        </div>
      </div>
      
      <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-8 text-center backdrop-blur-sm">
        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-indigo-500/10 text-indigo-400 mb-4">
          <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-slate-200">No active agents found</h3>
        <p className="text-sm text-slate-400 mt-2 max-w-sm mx-auto">
          Agents you register in your Go backend will be listable and manageable from this control center.
        </p>
      </div>
    </div>
  );
}
