import type { IconName } from "~/shared/components/Icon";

/** An entry in the app switcher. Each feature exposes one in its own app.ts. */
export type AppDef = {
  id: string;
  name: string;
  path: string;
  icon: IconName;
  enabled: boolean;
};
