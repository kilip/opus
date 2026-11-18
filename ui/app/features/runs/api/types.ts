import type { components } from "~/shared/api/schema";

type Schemas = components["schemas"];

export type RunItem = Schemas["domain.RunResponse"];
export type RunState = Schemas["domain.RunState"];
export type RunMode = Schemas["domain.RunMode"];
export type CancelRunRequest = Schemas["domain.CancelRunRequest"];
