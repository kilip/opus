import withSerwist from "@serwist/next";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  turbopack: {},
};

export default withSerwist({
  swSrc: "sw.ts",
  swDest: "public/sw.js",
})(nextConfig);
