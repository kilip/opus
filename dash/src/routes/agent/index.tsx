import { createFileRoute } from '@tanstack/react-router';
import { Bot, Plus } from 'lucide-react';
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/shared/components/ui';

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
    <div className="space-y-8">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2">
            <Badge variant="success">System ready</Badge>
          </div>
          <h1 className="font-sans">Agents</h1>
          <p className="mt-2 max-w-xl font-serif text-muted">
            Manage autonomous AI agents and supervise their active pipelines
            from one warm, focused control center.
          </p>
        </div>
        <Button type="button" className="w-full shrink-0 sm:w-auto">
          <Plus className="h-4 w-4" aria-hidden />
          New agent
        </Button>
      </header>

      <div className="editorial-rule" aria-hidden />

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-card bg-brand-primary/10 text-brand-primary">
              <Bot className="h-5 w-5" aria-hidden />
            </span>
            Active agents
          </CardTitle>
          <CardDescription>
            Agents registered in your Opus backend appear here for listing and
            management.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center rounded-card border border-dashed border-border bg-subtle/30 px-6 py-14 text-center">
            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-brand-primary/10 text-brand-primary">
              <Bot className="h-7 w-7" aria-hidden />
            </div>
            <p className="font-sans text-base font-medium text-foreground">
              No active agents yet
            </p>
            <p className="mt-2 max-w-sm font-serif text-sm text-muted">
              Connect your server and register agents to see them orchestrated
              in this workspace.
            </p>
          </div>
        </CardContent>
        <CardFooter className="justify-center sm:justify-start">
          <Button type="button" variant="secondary">
            View documentation
          </Button>
          <Button type="button" variant="ghost">
            Refresh list
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}
