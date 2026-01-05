# 变更日志

## 2026-01-05
- 将迁移方向调整为：`MartialBE/one-hub`(源) -> `songquanpeng/one-api`(目标)
- 更新通道类型映射逻辑：one-hub 的 `ChannelType*` 映射到 one-api 的 `channeltype.*`
- 环境变量改为 `ONEAPI_SOURCE_SQL_DSN` / `ONEAPI_TARGET_SQL_DSN`
- 同步更新文档与示例配置（README、docker-compose）
- 迁移健壮性增强：源库缺失某张表时直接提示并跳过（例如 one-hub 可能无 `abilities`）
- 新增迁移后置步骤：可选从目标库 `channels` 派生重建目标库 `abilities`（默认开启，可用 `ONEAPI_REBUILD_ABILITIES=false` 关闭）
- 修复 README 的 docker-compose 示例环境变量名（改为 SOURCE/TARGET）
- 修复 README 的 PostgreSQL DSN 示例格式（改为标准 `postgres://user:pass@host:5432/db?sslmode=...` 或 key-value 连接串）
- 更新部署配置：Dockerfile 环境变量名改为 SOURCE/TARGET，并增加 `ONEAPI_REBUILD_ABILITIES`；docker-compose 默认使用 `ghcr.io/lfglfg11/onehub-db-transfer:latest`
- 修复 DSN 解析：Postgres URL DSN 不再截断 `postgres://`；MySQL `mysql://` 支持解析并转换为 `go-sql-driver/mysql` DSN
- 修复 Postgres 兼容性：INSERT 使用 `$1..$n` 占位符，并按驱动正确引用表/列名（Postgres 使用 `"ident"`）
- 修复迁移列选择策略：仅迁移“源/目标同名列交集”，不再对目标缺失列强行填默认值，避免覆盖默认值/触发 NOT NULL/类型不匹配
- 移除阻塞暂停：删除 `fmt.Scanln()`，避免迁移结束后卡住
- 改进错误处理：表内扫描/插入/提交失败时回滚并跳过该表，不再 `log.Fatalf` 直接退出整个进程
