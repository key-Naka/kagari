# Kagari 服务器访问与部署

## 服务器访问

- 主机：`43.255.156.253`
- SSH 用户：`root`
- SSH 端口：`22`
- 专用部署私钥本机路径：`C:\Users\16327\.ssh\kagari_deploy_v2`
- 对应公钥指纹：`SHA256:wtUtGJ0++G2EEZ6Jy8tYhLlISFrncOjzJplGLZndEH8`
- 对应公钥标识：`kagari-deploy-v2`

使用 PowerShell 连接：

```powershell
ssh -i "$env:USERPROFILE\.ssh\kagari_deploy_v2" -o IdentitiesOnly=yes root@43.255.156.253
```

私钥内容和服务器密码不写入仓库、文档或聊天记录。私钥只保存在上面的本机路径；如需更换设备，应新建独立密钥并将其公钥添加到 `/root/.ssh/authorized_keys`。

## 已部署架构

- 应用目录：`/opt/kagari`
- 编排命令：`cd /opt/kagari && docker compose <command>`
- 私有生产环境文件：`/opt/kagari/.env`，权限为 `600`，不得下载、提交或复制到仓库。
- 主站：`https://ykagari.top`，Nginx 反向代理到 `127.0.0.1:3000`。
- API：`https://kagari-api.ykagari.top`，Nginx 反向代理到 `127.0.0.1:18080`。
- `api.ykagari.top` 属于既有项目，不能被 Kagari 覆盖。
- Kagari 容器：Nuxt Web、Go API、专属 MySQL 8.4 与 Redis 7.4；MySQL/Redis 不映射宿主机端口。
- Web 与 API 仅绑定宿主机回环地址，公网仅开放 Nginx 的 80/443。

## Nginx 与 HTTPS

- Nginx 配置源文件：
  - `/etc/nginx/kagari-conf/ykagari.top.conf`
  - `/etc/nginx/kagari-conf/kagari-api.ykagari.top.conf`
- 生效软链接位于 `/etc/nginx/conf.d/`。
- 两个域名均使用 Let's Encrypt 证书：`/etc/letsencrypt/live/ykagari.top/`。
- 证书覆盖：`ykagari.top`、`kagari-api.ykagari.top`。
- HTTP 会自动跳转 HTTPS；ACME 验证目录为 `/var/www/kagari-acme`。
- 证书到期日：`2026-11-08`。
- 自动续期任务：`/etc/cron.d/kagari-certbot-renew`，每日 `03:17` 执行 `certbot renew`；成功续期后自动重载 Nginx。

检查配置和续期：

```bash
nginx -t
certbot certificates
certbot renew --dry-run
```

## 常用部署与验证命令

上传新代码后，在服务器执行：

```bash
cd /opt/kagari
docker compose up -d --build
docker compose ps
```

查看日志：

```bash
cd /opt/kagari
docker compose logs -f --tail=100 frontend backend
```

验证服务：

```bash
curl -fsSI http://127.0.0.1:3000/
curl -fsS http://127.0.0.1:18080/health
curl -fsSI https://ykagari.top/
curl -fsS https://kagari-api.ykagari.top/health
```

API 正常时应返回：

```json
{"status":"ok","dependencies":{"mysql":"ok","redis":"ok"}}
```

## 已确认的服务器现状

- 系统：CentOS 7，根分区可用空间约 55GB。
- Docker：26.1.4；Docker Compose：v2.27.1。
- 1Panel 与 Nginx 正常运行。
- 已有 MySQL 8.4 容器，仅绑定 `127.0.0.1:13306`。
- 已有 Redis 7.4 容器，仅绑定 `127.0.0.1:16379`。
- 已有 `api.ykagari.top` HTTPS 虚拟主机，代理到 `127.0.0.1:8080`；该端口属于既有项目。

## 安全收口

1. 在云安全组中仅允许可信公网 IP 访问 22 端口。
2. 保留 `kagari-deploy-v2` 公钥，删除不再使用的旧部署公钥。
3. 轮换曾在聊天中暴露的 root 密码。
4. 安装并启用 Fail2ban，或在云侧启用 SSH 暴力破解防护。
5. 不要提交 `.env`、私钥、证书私钥或容器数据卷。
