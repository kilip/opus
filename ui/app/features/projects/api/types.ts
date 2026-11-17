import type { components } from "~/shared/api/schema";

type Schemas = components["schemas"];

export type Project = Schemas["domain.Project"];
export type CreateProjectRequest = Schemas["domain.CreateProjectRequest"];
export type UpdateProjectRequest = Schemas["domain.UpdateProjectRequest"];
export type ProjectNodeResponse = Schemas["domain.ProjectNodeResponse"];
export type UpdateProjectNodePathRequest =
	Schemas["domain.UpdateProjectNodePathRequest"];
