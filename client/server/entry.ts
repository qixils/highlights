import { apply, serve } from "@photonjs/hono";
import { Hono } from "hono";

const port = process.env.PORT ? parseInt(process.env.PORT, 10) : 30101;

export default startApp() as unknown;

function startApp() {
  const app = new Hono();

  apply(app);

  return serve(app, {
    port,
  });
}
