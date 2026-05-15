# Stage 1: Build
FROM node:22-alpine AS builder

WORKDIR /app

# Install pnpm with specific version
RUN npm install -g pnpm@11.1.2

# Copy package files and pnpm config
COPY package.json pnpm-lock.yaml* .npmrc* pnpm-workspace.yaml* ./

# Install with build scripts allowed for specific packages
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Set build environment variables
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# Build the application
RUN pnpm build

# Stage 2: Runtime
FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV production

# Copy necessary files from builder
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

CMD ["node", "server.js"]
