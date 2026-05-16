import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { logger } from "@/lib/logger";

/**
 * Proxy middleware for Next.js 16.
 * Handles route protection and session verification.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const refreshToken = request.cookies.get("refresh_token")?.value;

  logger.info(`[Proxy] Request: ${pathname} | Has Token: ${!!refreshToken}`);

  const isAuthRoute = pathname === "/login" || pathname.startsWith("/auth");
  const isPublicFile =
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") ||
    pathname === "/favicon.ico";

  // 3. Logic: If trying to access protected route without a session, go to login
  if (!isAuthRoute && !isPublicFile && !refreshToken) {
    logger.info(`[Proxy] Redirecting to /login from ${pathname} (No Token)`);
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // 4. Logic: If already logged in, don't show login page
  if (pathname === "/login" && refreshToken) {
    logger.info(`[Proxy] Redirecting to / from /login (Token present)`);
    return NextResponse.redirect(new URL("/", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    "/((?!api|_next/static|_next/image|favicon.ico).*)",
  ],
};
