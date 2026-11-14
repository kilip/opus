import { tasksApp } from "~/features/tasks/app";
import type { AppDef } from "~/shared/types";

/**
 * App switcher registry. Real features contribute their own `AppDef`;
 * disabled entries are placeholders for features that do not exist yet.
 */
export const apps: AppDef[] = [
  tasksApp,
  {
    id: "agents",
    name: "Agents",
    path: "/agents",
    icon: "bot",
    enabled: false,
  },
  { id: "runs", name: "Runs", path: "/runs", icon: "activity", enabled: false },
  {
    id: "settings",
    name: "Settings",
    path: "/settings",
    icon: "settings",
    enabled: false,
  },
];
