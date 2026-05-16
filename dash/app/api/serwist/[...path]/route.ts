import { createSerwistRoute } from "@serwist/turbopack";
import type { NextRequest, NextResponse } from "next/server";

const handler = createSerwistRoute({
  swSrc: "sw.ts",
  // biome-ignore lint/suspicious/noExplicitAny: library type mismatch
} as any);

export const GET = async (
  req: NextRequest,
  context: { params: Promise<{ path: string[] }> },
) => {
  const params = await context.params;
  return handler.GET(req, {
    params: Promise.resolve({ path: params.path.join("/") }),
  }) as Promise<NextResponse<unknown>>;
};

export const POST = GET;

export const generateStaticParams = async () => {
  const params = await handler.generateStaticParams();
  return params.map((p) => ({ path: p.path.split("/") }));
};
