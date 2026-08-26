# --- сборка ---
FROM node:22-alpine AS build
WORKDIR /app
# Слой зависимостей кешируется отдельно: package.json меняется реже исходников.
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# --- раздача ---
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
