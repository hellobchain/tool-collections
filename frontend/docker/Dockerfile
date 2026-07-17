# ---- Production Stage ----
FROM nginx:stable-alpine

# Nginx 配置
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# 构建产物
COPY dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
