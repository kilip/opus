import {
  AlertCircle,
  CheckCircle2,
  Info,
  Palette,
  TriangleAlert,
} from 'lucide-react';
import { useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Checkbox,
  Code,
  Field,
  Heading,
  Input,
  Select,
  Separator,
  Spinner,
  Switch,
  Text,
  Textarea,
} from '@/shared/components/ui';
import { cn } from '@/shared/lib/utils';
import { DemoSection } from './DemoSection';
import { ShowcasePanel } from './ShowcasePanel';

const navSections = [
  { id: 'typography', label: 'Typography' },
  { id: 'buttons', label: 'Buttons' },
  { id: 'badges', label: 'Badges' },
  { id: 'forms', label: 'Forms' },
  { id: 'controls', label: 'Controls' },
  { id: 'cards', label: 'Cards' },
  { id: 'feedback', label: 'Feedback' },
  { id: 'layout', label: 'Layout' },
] as const;

const colorSwatches = [
  { name: 'brand-dark', className: 'bg-brand-dark' },
  { name: 'brand-light', className: 'bg-brand-light' },
  { name: 'brand-primary', className: 'bg-brand-primary' },
  { name: 'brand-secondary', className: 'bg-brand-secondary' },
  { name: 'brand-muted', className: 'bg-brand-muted' },
  { name: 'brand-subtle', className: 'bg-brand-subtle' },
  { name: 'brand-success', className: 'bg-brand-success' },
] as const;

/**
 * Full component library showcase for Opus design system.
 */
export function DemoPage() {
  const [notifyEnabled, setNotifyEnabled] = useState(true);
  const [autoRun, setAutoRun] = useState(false);

  return (
    <div className="demo-page space-y-16 pb-20">
      <header className="demo-reveal relative overflow-hidden rounded-card border border-border bg-card p-8 shadow-card sm:p-10">
        <div className="pointer-events-none absolute -top-24 -right-16 h-56 w-56 rounded-full bg-brand-primary/15 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-20 -left-10 h-48 w-48 rounded-full bg-brand-secondary/10 blur-3xl" />

        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <Badge variant="outline" className="gap-1.5">
              <Palette className="h-3 w-3" aria-hidden />
              Charcoal &amp; Rust
            </Badge>
            <Heading level={1}>Opus design system</Heading>
            <Text className="text-lg">
              Premium, reusable primitives aligned with{' '}
              <Code>docs/BRANDING.md</Code>. Import from{' '}
              <Code>@/shared/components/ui</Code> — customize via variants and{' '}
              <Code>className</Code>, not one-off styles.
            </Text>
          </div>

          <nav
            aria-label="Section navigation"
            className="flex flex-wrap gap-2 lg:max-w-xs lg:justify-end"
          >
            {navSections.map((item) => (
              <a
                key={item.id}
                href={`#${item.id}`}
                className={cn(
                  'rounded-btn border border-border bg-subtle/50 px-3 py-1.5',
                  'font-sans text-xs font-medium text-muted',
                  'transition-all duration-200 hover:border-brand-primary/30 hover:text-foreground',
                )}
              >
                {item.label}
              </a>
            ))}
          </nav>
        </div>
      </header>

      <DemoSection
        id="typography"
        title="Typography"
        description="Poppins for structure, Lora for editorial warmth, JetBrains Mono for technical surfaces."
      >
        <div className="grid gap-4 lg:grid-cols-2">
          <ShowcasePanel label="Headings">
            <div className="w-full space-y-3">
              <Heading level={1}>Page title</Heading>
              <Heading level={2}>Section title</Heading>
              <Heading level={3}>Sub-section</Heading>
              <Heading level={4}>Card heading</Heading>
            </div>
          </ShowcasePanel>
          <ShowcasePanel label="Body & code">
            <div className="w-full space-y-3">
              <Text>
                Body copy uses the serif stack for long-form readability and a
                calm, book-like rhythm.
              </Text>
              <Text variant="muted">
                Muted text for descriptions, metadata, and secondary context.
              </Text>
              <p>
                Agent ID: <Code>agt_7f2k9m1p</Code>
              </p>
            </div>
          </ShowcasePanel>
        </div>

        <ShowcasePanel label="Color tokens" className="lg:col-span-2">
          <div className="grid w-full grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
            {colorSwatches.map((swatch) => (
              <div key={swatch.name} className="space-y-2 text-center">
                <div
                  className={cn(
                    'mx-auto h-12 w-full rounded-card border border-border shadow-sm',
                    swatch.className,
                  )}
                />
                <span className="font-mono text-[10px] text-muted">
                  {swatch.name}
                </span>
              </div>
            ))}
          </div>
        </ShowcasePanel>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="buttons"
        title="Buttons"
        description="Primary actions, secondary options, and low-emphasis controls with consistent motion."
      >
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <ShowcasePanel label="Variants">
            <Button type="button">Primary</Button>
            <Button type="button" variant="secondary">
              Secondary
            </Button>
            <Button type="button" variant="ghost">
              Ghost
            </Button>
            <Button type="button" variant="outline">
              Outline
            </Button>
            <Button type="button" variant="destructive">
              Destructive
            </Button>
          </ShowcasePanel>
          <ShowcasePanel label="Sizes">
            <Button type="button" size="sm">
              Small
            </Button>
            <Button type="button" size="md">
              Medium
            </Button>
            <Button type="button" size="lg">
              Large
            </Button>
          </ShowcasePanel>
          <ShowcasePanel label="States">
            <Button type="button" disabled>
              Disabled
            </Button>
            <Button type="button" variant="secondary" disabled>
              Disabled
            </Button>
          </ShowcasePanel>
        </div>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="badges"
        title="Badges"
        description="Compact status labels for connectivity, alerts, and neutral metadata."
      >
        <ShowcasePanel label="Variants">
          <Badge variant="success">Online</Badge>
          <Badge variant="alert">Attention</Badge>
          <Badge variant="muted">Draft</Badge>
          <Badge variant="outline">Neutral</Badge>
        </ShowcasePanel>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="forms"
        title="Forms"
        description="Editorial inputs with Field wrappers for labels, hints, and validation."
      >
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Contact</CardTitle>
              <CardDescription>
                Standard inputs use serif text; focus rings use brand primary.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <Field
                label="Display name"
                htmlFor="demo-name"
                hint="Shown in agent logs and notifications."
                required
              >
                <Input id="demo-name" placeholder="Research assistant" />
              </Field>
              <Field
                label="Model tier"
                htmlFor="demo-tier"
                error={
                  !notifyEnabled ? 'Select a tier to continue.' : undefined
                }
              >
                <Select id="demo-tier" defaultValue="standard">
                  <option value="standard">Standard</option>
                  <option value="fast">Fast</option>
                  <option value="deep">Deep reasoning</option>
                </Select>
              </Field>
              <Field label="Instructions" htmlFor="demo-prompt">
                <Textarea
                  id="demo-prompt"
                  placeholder="Describe how this agent should behave…"
                  rows={4}
                />
              </Field>
            </CardContent>
            <CardFooter>
              <Button type="button">Save agent</Button>
              <Button type="button" variant="secondary">
                Cancel
              </Button>
            </CardFooter>
          </Card>

          <ShowcasePanel label="Input states">
            <div className="grid w-full gap-4">
              <Input placeholder="Default" aria-label="Default input" />
              <Input
                placeholder="Disabled"
                disabled
                aria-label="Disabled input"
              />
              <Input defaultValue="With value" aria-label="Filled input" />
            </div>
          </ShowcasePanel>
        </div>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="controls"
        title="Controls"
        description="Checkboxes and switches for preferences and feature toggles."
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <ShowcasePanel label="Checkbox">
            <Checkbox label="Enable proactive evaluations" defaultChecked />
            <Checkbox label="Share anonymized telemetry" />
          </ShowcasePanel>
          <ShowcasePanel label="Switch">
            <Switch
              label="Push notifications"
              checked={notifyEnabled}
              onCheckedChange={setNotifyEnabled}
            />
            <Switch
              label="Auto-run on vault write"
              checked={autoRun}
              onCheckedChange={setAutoRun}
            />
          </ShowcasePanel>
        </div>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="cards"
        title="Cards"
        description="Content containers with optional hover lift for interactive surfaces."
      >
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Static card</CardTitle>
              <CardDescription>
                Default panel for grouped content and forms.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Text variant="muted">
                Warm surfaces, subtle shadow, and rounded-card corners per brand
                guidelines.
              </Text>
            </CardContent>
          </Card>
          <Card interactive>
            <CardHeader>
              <CardTitle>Interactive card</CardTitle>
              <CardDescription>
                Lifts on hover — use for clickable list items or links.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Badge variant="success">Ready</Badge>
            </CardContent>
          </Card>
        </div>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="feedback"
        title="Feedback"
        description="Alerts and spinners for system status and async operations."
      >
        <div className="grid gap-4">
          <ShowcasePanel label="Alerts" className="[&_figure]:w-full">
            <div className="grid w-full gap-3">
              <Alert
                variant="info"
                title="Configuration saved"
                icon={<Info className="h-4 w-4" aria-hidden />}
              >
                <p>
                  Your agent will pick up changes on the next evaluation cycle.
                </p>
              </Alert>
              <Alert
                variant="success"
                title="Integration connected"
                icon={<CheckCircle2 className="h-4 w-4" aria-hidden />}
              >
                <p>Gmail sync is active and polling every five minutes.</p>
              </Alert>
              <Alert
                variant="warning"
                title="Rate limit approaching"
                icon={<TriangleAlert className="h-4 w-4" aria-hidden />}
              >
                <p>Consider reducing concurrent agent evaluations.</p>
              </Alert>
              <Alert
                variant="error"
                title="Job failed"
                icon={<AlertCircle className="h-4 w-4" aria-hidden />}
              >
                <p>
                  Workflow step &quot;summarize inbox&quot; timed out after
                  120s.
                </p>
              </Alert>
            </div>
          </ShowcasePanel>
          <ShowcasePanel label="Spinner">
            <Spinner size="sm" />
            <Spinner size="md" />
            <Spinner size="lg" />
          </ShowcasePanel>
        </div>
      </DemoSection>

      <div className="editorial-rule" aria-hidden />

      <DemoSection
        id="layout"
        title="Layout"
        description="Separators and rules for structuring dense dashboards."
      >
        <ShowcasePanel label="Separator">
          <div className="flex w-full flex-col gap-4">
            <Text variant="muted">Above the rule</Text>
            <Separator />
            <Text variant="muted">Below the rule</Text>
            <div className="flex h-12 items-stretch gap-4">
              <span className="font-serif text-sm text-muted">Left</span>
              <Separator orientation="vertical" />
              <span className="font-serif text-sm text-muted">Right</span>
            </div>
          </div>
        </ShowcasePanel>
      </DemoSection>
    </div>
  );
}
