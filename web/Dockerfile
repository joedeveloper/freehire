# Production image for the freehire SPA. nginx serves the static build and
# proxies /api to the backend container, keeping the SPA and API same-origin
# (required for the SameSite=Lax auth cookie — see AGENT.md).
#
# Build context is the web/ directory:
#   docker build -f web/Dockerfile web/

# --- build stage ---
FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# --- runtime stage ---
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
