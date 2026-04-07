# icmp-ddns 控制面板 — 完整架构文档

版本：1.0

## 一、目标

- 提供一个可管理的控制面板，用于查看与修改 `icmp-ddns` 的运行配置、查看运行状态、审计操作和回滚配置。
- 兼顾轻量与安全：默认单机部署，支持容器化/K8s 部署；提供最小认证方案与可选扩展。

## 二、关键需求

- 配置查看与修改（YAML/JSON）。
- 安全认证与权限（至少管理员账号/Token）。
- 配置变更的版本化与回滚。
- 日志与审计：记录谁在什么时候做了什么改动。
- 健康检查与指标导出（Prometheus）。
- 简易前端（SPA）用于操作，能在低权限环境下运行。

## 三、总体架构概览

- Browser (管理者) → 静态前端 (`./static`) → 管理 API (`/admin/*`) → 主进程（内存 cfg + 持久化文件）
- 主进程负责：与 eBPF 读取事件、决定是否更新 Cloudflare、发起通知。
- 可选组件：反向代理（Nginx/Caddy）、认证服务、远程日志/监控（ELK/Prometheus）

```mermaid
graph LR
  Browser[Browser] -->|HTTP(S)| Proxy[Reverse Proxy (nginx/caddy)]
  Proxy --> AdminUI[Static Frontend]
  Proxy --> API[icmp-ddns Admin API]
  API --> Service[icmp-ddns Service]
  Service --> ConfigFile[config.yaml]
  Service --> EBPF[eBPF kernel program]
  Service --> Cloudflare[Cloudflare API]
  Service --> Notify[WeCom / Webhook]
  API --> Logs[Audit & App Logs]
  Metrics[Prometheus] -->|scrape| API
```

## 四、配置存储与版本化

- 主配置文件路径：`/etc/icmp-ddns/config.yaml`（当前实现）。
- 建议增加配置备份目录：`/etc/icmp-ddns/backups/`，保存 `config-YYYYMMDD-HHMMSS.yaml`，保留最近 N 份。
- 保存策略：原子写入（写入临时文件再重命名），保存成功后触发一次内存重载。

示例：保存流程

1. API 接收到配置变更请求并校验。
2. 将当前 `config.yaml` 复制到 `backups/`（带时间戳）。
3. 将新配置写入临时文件 `.tmp` 并 `rename` 覆盖 `config.yaml`。
4. 调用内存重载逻辑（channel 或 SIGHUP）。

## 五、API 设计（推荐 REST + JSON）

认证：推荐使用短期管理 Token（Bearer token）或 Basic auth（仅内网）。生产环境推荐在反向代理做 TLS 与认证/IDP。

主要接口：

- `GET /admin/health` — 返回服务健康信息（200/500）。
- `GET /admin/config` — 返回当前配置（需鉴权，敏感字段可掩码）。
- `POST /admin/config` — 提交完整配置（JSON），服务器校验并持久化，返回 `204 No Content`。
- `POST /admin/config/validate` — 校验配置但不保存，返回校验结果。
- `POST /admin/reload` — 从磁盘重新加载配置（管理员）。
- `POST /admin/restart` — 请求服务重启（需进程管理器支持）。
- `GET /admin/logs?tail=200` — 获取最近日志（管理员，需限流）。
- `GET /metrics` — Prometheus 指标（可选，公开给内部监控）。

示例：`GET /admin/config` 返回 JSON（敏感字段 `token` 应返回 `*****`，前端可选择显示/编辑）。

OpenAPI / Swagger：建议为上述接口添加简单 OpenAPI 描述，便于生成文档与前端 Mock。

## 六、认证与权限（建议）

方案分级：

1. 最小可用：单个静态管理 Token（在 `config.yaml` 或环境变量中）；API 要求 `Authorization: Bearer <token>`。
2. 推荐（生产）：反向代理处理 TLS + 基于 IP 的访问控制 + OAuth2/OIDC（如果已有 IDP）。
3. 角色：`admin`（读写、重载、查看日志）、`viewer`（只读）。

密钥与存储：

- 不要将明文长期放在磁盘上；可使用操作系统密钥库或外部 Secret Manager。
- 对 `config.yaml` 中的敏感字段（如 Cloudflare Token）进行掩码显示与可选加密。

## 七、审计与日志

- 审计日志内容：时间、操作者（token/用户名）、操作类型（read/config_update/reload/restart）、变更前后摘要、IP。
- 写入：`/var/log/icmp-ddns-admin.log`（结构化 JSON 行式），并和应用日志分开。
- 保留策略：转储到远端集中日志或循环切分（logrotate）。

示例审计条目（JSON）:

```json
{"ts":"2026-04-07T12:00:00Z","user":"admin","ip":"1.2.3.4","action":"config_update","backup":"config-20260407-120000.yaml"}
```

## 八、前端设计（静态 SPA）

- 路由建议：
  - `/` 仪表盘（健康、最后更新、当前 IP）
  - `/config` 配置编辑与保存
  - `/logs` 日志查看（tail）
  - `/audit` 审计记录
  - `/settings` 管理 Token 与备份

- 前端功能要点：表单校验、敏感字段掩码、保存后提示并触发后端校验、显示备份历史与回滚按钮。

## 九、健康检查与监控

- `GET /admin/health`：包含 `uptime`、`last_update`、`last_cf_update` 等字段。
- Prometheus 指标：`icmp_ddns_updates_total`, `icmp_ddns_update_errors_total`, `icmp_ddns_last_update_timestamp`。

## 十、部署方案（建议三种）

1) 单机 systemd（轻量、简单）

示例 `systemd` unit:
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

2) Docker Compose（便于打包）

docker-compose.yml 中将 `config.yaml` 作为卷挂载，使用 `env_file` 管理敏感变量。

3) Kubernetes

- 将配置分为 `ConfigMap`（非敏感）和 `Secret`（Token），用 `Deployment` + `Pod`，并通过 `Ingress` 做 TLS。

## 十一、安全加固 & 注意事项

- 强制 TLS（至少在反向代理层面）。
- 管理 API 仅对受信任网络或经过身份验证的用户开放。
- 对上传的配置进行严格校验（防止注入/非法字段）。
- 对敏感字段在 UI/日志中掩码。日志勿输出完整 token。
- 添加速率限制与 IP 黑名单以防滥用。

## 十二、回滚与备份

- 每次保存前自动备份旧配置；提供 `/admin/backups` 列表与回滚接口（管理员确认）。

## 十三、运维 Runbook（简要）

- 查看状态：`systemctl status icmp-ddns` 或 `docker logs --tail 200 icmp-ddns`。
- 强制重载：`curl -X POST -H "Authorization: Bearer <token>" http://localhost:8080/admin/reload`。
- 回滚配置：通过后台备份选择旧配置并 `POST /admin/config`。

## 十四、实现清单（优先级）

1. 完成 API 验证中间件（Token 验证、IP 白名单）。
2. 完成 `GET/POST /admin/config`（带备份与原子写入）。
3. 增加审计日志写入逻辑。 
4. 在前端完成配置表单与备份回滚 UI。 
5. 添加 `GET /admin/logs`（限流）与 `GET /admin/health`。 
6. 添加 Prometheus metrics（可选）。
7. 编写 systemd / docker-compose / k8s 示例并更新 README。

## 十五、示例 OpenAPI（简短示意）

```yaml
openapi: 3.0.0
paths:
  /admin/config:
    get:
      security:
        - bearerAuth: []
      responses:
        '200':
          description: current config
    post:
      security:
        - bearerAuth: []
      requestBody:
        required: true
      responses:
        '204':
          description: saved

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
```

---

## 附：下一步我可以立刻做的事

- 生成完整 OpenAPI 文件并把 `admin.go` 的路由调整为与之匹配。
- 为 `admin.go` 添加鉴权中间件（token 验证）并在前端加登录/提示。
- 增加备份/回滚 API 与审计日志写入。

如果你同意，我会先实现**认证中间件 + POST/GET /admin/config 完整实现与备份/回滚 API**。请选择：

- 现在开始实现认证与配置接口
- 先生成 OpenAPI 描述与 API Mock
- 先生成 systemd / docker-compose / k8s 部署示例
