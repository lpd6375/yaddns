# yaddns

icmp-ddns — 基于 eBPF 的 ICMP 触发 DDNS 更新服务，带简易管理面板。

主要功能

- 监听 ICMP 源地址事件并触发 Cloudflare DDNS 更新。
- 内置管理面板：查看/修改配置、备份与回滚、查看日志与审计记录、健康检查与 Prometheus 指标。

快速开始（本地开发）

1. 安装 Go 1.20+。
2. 克隆仓库并进入目录。

```bash
git clone git@github.com:lpd6375/yaddns.git
cd yaddns
```

3. 本项目包含对 Linux eBPF 的支持；如果在非 Linux 平台运行，代码会使用一个 stub（不加载 eBPF），仍然可以启动管理面板用于调试。

构建并运行：

```bash
go mod tidy
go run .
```

管理面板

- 访问: `http://localhost:8080/`（默认）。
- API 文档（OpenAPI）: `http://localhost:8080/openapi.yaml`。

通知集成

- 企业微信（Webhook）: 在 `config` 的 `notify.wecom_webhook` 填写 webhook 地址，勾选 `enable_wecom` 开启。
- Telegram: 在 `notify.telegram_bot_token` 填写 bot token（格式 `123456:ABC-DEF...`），在 `notify.telegram_chat_id` 填写 chat id，勾选 `enable_telegram` 开启。


配置文件

- 默认路径: `/etc/icmp-ddns/config.yaml`。
- 本地调试可修改 `configPath` 常量或以管理员权限创建该文件。

示例 systemd unit

```ini
[Unit]
Description=icmp-ddns
After=network.target

[Service]
ExecStart=/usr/local/bin/icmp-ddns
Restart=on-failure
User=ddns

[Install]
WantedBy=multi-user.target
```

示例 docker-compose（简要）

```yaml
version: '3'
services:
	yaddns:
		image: yourrepo/icmp-ddns:latest
		volumes:
			- ./config.yaml:/etc/icmp-ddns/config.yaml:ro
		network_mode: host
		restart: unless-stopped
```

注意事项

- 生产环境请务必为管理接口配置 `runtime.admin_token` 或在反向代理层使用 TLS 与身份验证。
- 审计与日志默认写入 `/var/log/icmp-ddns-admin.log` 与 `/var/log/icmp-ddns.log`，请确保写权限。

更多文档

见 `CONTROL_PANEL_ARCHITECTURE.md` 获取控制面板架构、API 设计与部署建议。
