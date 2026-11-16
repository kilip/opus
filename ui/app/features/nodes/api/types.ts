import type { components } from "~/shared/api/schema";

type Schemas = components["schemas"];

export type NodeItem = Schemas["domain.NodeResponse"];
export type Role = Schemas["domain.Role"];
export type Status = Schemas["domain.Status"];
export type JoinTokenResponse = Schemas["domain.JoinTokenResponse"];
