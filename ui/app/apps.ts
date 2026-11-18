import { nodesApp } from "~/features/nodes/app";
import { projectsApp } from "~/features/projects/app";
import { runsApp } from "~/features/runs/app";
import { tasksApp } from "~/features/tasks/app";
import type { AppDef } from "~/shared/types";

/**
 * App switcher registry. Real features contribute their own `AppDef`;
 * disabled entries are placeholders for features that do not exist yet.
 */
export const apps: AppDef[] = [
	tasksApp,
	projectsApp,
	nodesApp,
	{
		id: "agents",
		name: "Agents",
		path: "/agents",
		icon: "bot",
		enabled: false,
	},
	runsApp,
	{
		id: "settings",
		name: "Settings",
		path: "/settings",
		icon: "settings",
		enabled: false,
	},
];
