import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { logger } from "@/lib/logger";

/**
 * Proxy middleware for Next.js 16.
 * Handles route protection and session verification.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 1. Get the refresh token from cookies
  const refreshToken = request.cookies.get("refresh_token");

  // 2. Define paths
  const isAuthRoute = pathname === "/login" || pathname.startsWith("/auth");
  const isPublicFile =
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") ||
    pathname === "/favicon.ico";

  // 3. Logic: If trying to access protected route without a session, go to login
  if (!isAuthRoute && !isPublicFile && !refreshToken) {
    logger.info(
      `[Proxy] Unauthorized access to ${pathname}, redirecting to /login`,
    );
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // 4. Logic: If already logged in, don't show login page
  if (pathname === "/login" && refreshToken) {
    logger.info(`[Proxy] Already logged in, redirecting from /login to /`);
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
