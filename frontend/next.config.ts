import type { NextConfig } from "next";
const config: NextConfig = {
  reactStrictMode: true,
  agentRules: false,
  async rewrites() {
    const base = process.env.NEXT_PUBLIC_API_URL;
    if (!base)
      throw new Error(
        "NEXT_PUBLIC_API_URL is required; run with make frontend or make verify",
      );
    return [{ source: "/api/:path*", destination: `${base}/api/:path*` }];
  },
};
export default config;
