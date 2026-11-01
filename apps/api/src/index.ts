import { auth } from "@opus/auth";
import { Elysia } from "elysia";
import { apiEnv } from "./config";

export const app = new Elysia()
  .mount(auth.handler)
  .get("/", () => ({ name: "opus-api", status: "ok" }))
  .get("/api/me", async ({ request, set }) => {
    const session = await auth.api.getSession({
      headers: request.headers,
    });
    if (!session) {
      set.status = 401;
      return { error: "Unauthorized" };
    }
    return session;
  });

if (import.meta.main) {
  app.listen(apiEnv.PORT);
  console.log(`🦊 Elysia is running at http://${app.server?.hostname}:${app.server?.port}`);
}

export type App = typeof app;
