import type { components } from "~/shared/api/schema";

/*
 * API types for the tasks feature. They come straight from the generated
 * OpenAPI schema; this file is the only place that knows the schema names, so
 * a rename in the spec (e.g. after re-running swag) is a one-file fix.
 */
type Schemas = components["schemas"];

export type Task = Schemas["handler.Task"];
export type TaskList = Schemas["handler.TaskList"];
export type TaskStatus = Schemas["handler.TaskStatus"];
export type TaskPriority = Schemas["handler.TaskPriority"];
export type TaskSource = Schemas["handler.TaskSource"];
export type TaskType = Schemas["handler.TaskType"];
export type Assignee = Schemas["handler.Assignee"];
export type Activity = Schemas["handler.Activity"];
export type ActivityKind = Schemas["handler.ActivityKind"];
export type ActivityList = Schemas["handler.ActivityList"];
