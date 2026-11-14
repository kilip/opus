import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const tasksRoutes: RouteConfigEntry[] = [
  route("tasks", "features/tasks/pages/TasksPage.tsx"),
  // `:taskId` is either a task id or the literal "new".
  route("tasks/:taskId", "features/tasks/pages/TaskFormPage.tsx"),
];
