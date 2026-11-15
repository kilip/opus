import type { components } from "~/shared/api/schema";

/*
 * API types for the tasks feature.
 * Aliases of the generated OpenAPI schema components["schemas"].
 */
type Schemas = components["schemas"];

export type Task = Schemas["domain.Task"];
export type TaskStatus = Schemas["domain.TaskStatus"];
export type TaskPriority = Schemas["domain.TaskPriority"];
export type TaskSource = Schemas["domain.TaskSource"];
export type TaskType = Schemas["domain.TaskType"];
export type Assignee = Schemas["domain.Assignee"];
export type Activity = Schemas["domain.Activity"];
export type ActivityKind = Schemas["domain.ActivityKind"];
export type CreateTaskRequest = Schemas["domain.CreateTaskRequest"];
export type UpdateTaskRequest = Schemas["domain.UpdateTaskRequest"];
export type CreateActivityRequest = Schemas["domain.CreateActivityRequest"];
