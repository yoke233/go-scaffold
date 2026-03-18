# go-scaffold

一个偏实战的 Go 脚手架，目标不是“把所有东西都抽象出来”，而是让你用
最少手改文件，稳定产出一个带 HTTP / gRPC、配置分层、代码生成、事务封装、
数据库迁移、自检和 OpenAPI 文档的项目骨架。

更完整的架构取舍放在 [design.md](./design.md)。

## 你会得到什么

- Kratos 双协议服务骨架：HTTP + gRPC
- Proto 驱动生成链路：pb / grpc / http / errors / OpenAPI
- GORM Gen + schema SQL 生成查询代码
- Wire 编译期依赖注入
- `UnitOfWork` 风格事务封装，方便跨 repo 组合写法
- 统一 JWT 认证、中间件注入、当前用户上下文
- 配置分层：`config.yaml` + `config.{env}.yaml` + 环境变量覆盖
- `scaffold add-feature` 新增业务域骨架
- `scaffold doctor` / `upgrade --check` 做最小治理闭环

## 快速开始

先做环境自检：

```bash
make doctor
```

安装工具、生成代码并跑测试：

```bash
make bootstrap
```

启动本地 PostgreSQL：

```bash
make db-up
```

启动服务：

```bash
make run
```

服务启动后可先检查：

- `GET /healthz`
- `GET /readyz`

## 常用命令

```bash
make help
make generate
make docs
make test
make lint
make ci
make proto-breaking
make upgrade-check
```

说明：

- `make generate` 会统一执行 proto、GORM、codegen、wire 生成
- `make docs` 会刷新 `docs/openapi/**/*.openapi.yaml` 文档产物
- `make ci` 对齐本地最小 CI 流程

## 新增业务域

新增一个 `order` 域：

```bash
make add-feature name=order
```

命令会补出这些骨架：

- `api/order/v1/*.proto`
- `internal/domain/ports/order.go`
- `internal/feature/order/facade.go`
- `internal/feature/order/wire_bind.go`
- `db/schema/orders.sql`
- `db/migrations/*_create_orders.sql`

随后执行：

```bash
make generate
make test
```

你通常只需要重点补这几类文件：

- `api/<feature>/v1/*.proto`
- `internal/feature/<feature>/usecase.go`
- `internal/feature/<feature>/repo.go`
- `db/migrations/*.sql`

## 配置加载规则

默认入口配置是 `configs/config.yaml`，加载顺序如下：

```text
configs/config.yaml
  -> configs/config.{env}.yaml
  -> 环境变量覆盖
```

其中 `env` 取值优先级：

1. `-env`
2. `APP_ENV`
3. `config.yaml` 里的 `app.env`

当前约定文件：

- `configs/config.yaml`：基础配置
- `configs/config.example.yaml`：示例配置
- `configs/config.local.yaml`：本地覆盖
- `configs/config.test.yaml`：测试覆盖

当前支持的环境变量覆盖：

- `APP_NAME`
- `APP_ENV`
- `APP_HTTP_ADDR`
- `APP_GRPC_ADDR`
- `APP_LOG_LEVEL`
- `APP_AUTH_JWT_ISSUER`
- `APP_AUTH_JWT_SIGNING_KEY`
- `APP_AUTH_JWT_ACCESS_TOKEN_TTL`
- `APP_DATABASE_DSN`

## 认证约定

当前脚手架默认使用 `JWT Bearer` 认证，HTTP 和 gRPC 共用一套认证中间件。

传递方式：

- HTTP：`Authorization: Bearer <access-token>`
- gRPC metadata：`authorization: Bearer <access-token>`

当前公开入口：

- `GET /healthz`
- `GET /readyz`
- `CreateUser`

当前用户上下文会注入最小主体：

- `user_id`
- `subject`
- `iat`
- `exp`

当前版本只提供认证底座，不包含：

- 登录 API
- 密码存储
- refresh token
- 多租户

## 文档与生成产物

Proto 生成配置在 `buf.gen.yaml`，当前会产出：

- `gen/`：Go 代码生成结果
- `cmd/server/features_gen.go`：服务注册聚合
- `cmd/server/wire_gen.go`：Wire 注入代码
- `docs/openapi/**/*.openapi.yaml`：按 proto 分文件的 OpenAPI 文档

如果你改了 proto、schema 或新增 feature，优先执行：

```bash
make generate
```

如果只想刷新接口文档：

```bash
make docs
```

## 升级治理最小版

根目录的 `scaffold.yaml` 记录了当前项目采用的脚手架版本，以及关键生成产物
路径。当前先做最小能力，不自动迁移，只做检查。

日常可用两个命令：

```bash
make doctor
make upgrade-check
```

它们分别解决的问题：

- `doctor`：检查必需文件、工具、生成产物、文档目录是否齐全
- `upgrade --check`：检查项目记录的 `scaffold_version` 是否与当前脚手架工具一致

如果检查提示缺失生成文件，通常按下面修复：

```bash
make generate
make docs
```

## CI 与工程约束

仓库当前内置这些基础约束：

- `.github/workflows/ci.yaml`
- `.golangci.yml`
- `buf lint`
- `buf breaking`

本地建议至少跑到这一步再提交：

```bash
make ci
```

## 还没覆盖的东西

这个脚手架现在已经能做“中小型业务服务起盘”，但还没把治理做满。下一批更值
得补的方向通常是：

- 鉴权与多租户约定
- 领域事件 / Outbox
- 更完整的可观测性：metrics / tracing / audit
- 集成测试容器编排
- 自动升级迁移脚本，而不只是 `--check`
