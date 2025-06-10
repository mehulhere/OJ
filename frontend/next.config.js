/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Disable App Router to use Pages Router exclusively
  experimental: {
    appDir: false,
  },
};

module.exports = nextConfig; 