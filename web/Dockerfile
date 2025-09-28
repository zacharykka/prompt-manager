# syntax=docker/dockerfile:1

FROM node:20-alpine AS build
WORKDIR /app

# 仅复制包管理文件以充分利用构建缓存
COPY package.json package-lock.json ./
RUN npm ci

# 拷贝剩余源码并构建
COPY . .
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM nginx:1.27-alpine AS runtime
WORKDIR /usr/share/nginx/html

# 替换默认站点配置，提供前端静态资源与 API 反向代理
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist ./

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -q --spider http://localhost/healthz || exit 1

# 简单静态健康检查响应
RUN printf 'ok' > /usr/share/nginx/html/healthz
