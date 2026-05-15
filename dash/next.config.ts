import withSerwist from "@serwist/next";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  output: "standalone",
  turbopack: {},
};

export default withSerwist({
  swSrc: "sw.ts",
  swDest: "public/sw.js",
})(nextConfig);
