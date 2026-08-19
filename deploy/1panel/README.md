# Kagari 1Panel 生产部署

本目录记录 1Panel 可管理的生产发布契约。Compose 只把前端与 API 绑定到宿主机回环地址；MySQL、Redis、宿主指标读取器和 Docker socket 代理仅在内部 Docker 网络中可达。

## 准备

1. 在 DNS 中把 `ykagari.top` 与 `kagari-api.ykagari.top` 指向服务器。
2. 确认证书 `/etc/letsencrypt/live/ykagari.top/fullchain.pem` 覆盖两个域名，并保留 `/var/www/kagari-acme` 供既有 Certbot webroot 续期任务使用。
3. 复制 `env.production.example` 为仓库根目录 `.env`，逐项替换所有 `replace-with-...`。用密码生成器创建互不复用的数据库、Redis 和管理员密码；不要把 `.env` 提交到 Git。
4. 仅开放公网 `80/tcp` 与 `443/tcp`。不要在防火墙中开放 3000、18080、3306、6379、8090 或 Docker socket proxy。

## 由 1Panel 编排

在 1Panel 的容器 / Compose 页面导入仓库根目录 `docker-compose.yml`，环境文件选择根目录 `.env`。等价的命令行预检与启动命令是：

```bash
docker compose --env-file .env config
docker compose --env-file .env up -d --build
docker compose ps
cd frontend && npm run build && npm run verify:production
```

通过 1Panel 的 OpenResty 高级配置，让 Nginx 的 `http` 上下文 include 完整的 `nginx/kagari.conf`，其中包含顶层 `map`、80 端口 ACME/HTTPS 跳转 server，以及两个 HTTPS server。不要只复制 HTTPS server 块，否则 `$connection_upgrade` 不会定义且证书续期路径会丢失。先运行 Nginx 配置检查，再重载；首次上线不要删除已有证书或站点配置。

## 验收与回滚

发布后依次检查：

```text
https://ykagari.top/health
https://kagari-api.ykagari.top/health
https://ykagari.top/robots.txt
https://ykagari.top/sitemap.xml
```

再执行公开浏览、历史动效、Mini Player、访客留言、管理员发布和媒体上传的 Playwright 流程。若健康检查或端到端流程失败，保留当前 `.env` 与数据卷，把 Compose 镜像恢复为上一个已验证版本后重新启动；不要删除 `mysql-data` 或 `redis-data` 卷。
