export const BASE_ERRORS: Record<string, string> = {
  RESOURCE_NOT_FOUND: "The requested resource was not found.",
  BAD_REQUEST: "The request was invalid.",
  UNAUTHORIZED: "You must be signed in to perform this action.",
  FORBIDDEN: "You do not have permission to perform this action.",
  CONFLICT: "The request conflicts with existing state.",
  INTERNAL_SERVER_ERROR: "An unexpected server error occurred.",
};

export function resolveErrorMessage(
  code: string,
  featureErrors?: Record<string, string>,
): string {
  return (
    featureErrors?.[code] ??
    BASE_ERRORS[code] ??
    (code || "An unknown error occurred.")
  );
}
