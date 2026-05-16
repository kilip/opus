import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { logger } from "@/lib/logger";

/**
 * Proxy middleware for Next.js 16.
 * Handles route protection and session verification.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const refreshToken = request.cookies.get("refresh_token");

  // Debug: Cek semua cookie yang ada
  const allCookies = request.cookies
    .getAll()
    .map((c) => c.name)
    .join(", ");
  logger.info(
    `[Proxy] Request: ${pathname} | Has refresh_token: ${!!refreshToken} | All Cookies: [${allCookies}]`,
  );

  const isAuthRoute = pathname === "/login" || pathname.startsWith("/auth");
  const isPublicFile =
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") ||
    pathname === "/favicon.ico";

  // 1. Jika user di halaman login
  if (pathname === "/login") {
    // Jika sudah punya token, arahkan ke home
    if (refreshToken) {
      return NextResponse.redirect(new URL("/", request.url));
    }
    // Jika belum, biarkan (untuk menampilkan halaman login)
    return NextResponse.next();
  }

  // 2. Jika bukan route auth/public, dan tidak ada token, arahkan ke login
  if (!isAuthRoute && !isPublicFile && !refreshToken) {
    logger.info(`[Proxy] Redirecting to /login from ${pathname} (No Token)`);
    return NextResponse.redirect(new URL("/login", request.url));
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
    "/((?!api|login|_next/static|_next/image|favicon.ico).*)",
  ],
};
