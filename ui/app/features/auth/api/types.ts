import type { components } from "~/shared/api/schema";

/*
 * API types for the auth feature.
 * Aliases of the generated OpenAPI schema components["schemas"].
 */
type Schemas = components["schemas"];

export type User = Schemas["domain.UserResponse"];
export type SetupStatus = Schemas["domain.SetupStatusResponse"];
export type OTPRequestBody = Schemas["domain.OTPRequestBody"];
export type OTPVerifyBody = Schemas["domain.OTPVerifyBody"];
