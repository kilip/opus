import type { components } from "~/shared/api/schema";

type Schemas = components["schemas"];

export type NodeItem = Schemas["domain.NodeResponse"];
export type TaskItem = Schemas["domain.Task"];
export type TaskStatus = Schemas["domain.TaskStatus"];
export type RunItem = Schemas["domain.RunResponse"];
export type RunState = Schemas["domain.RunState"];
