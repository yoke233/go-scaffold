# 项目架构设计

## 架构选型

| 层面 | 选型 | 职责 |
|---|---|---|
| **框架** | Kratos | HTTP/gRPC 双协议、路由注册、中间件、错误处理 |
| **Proto 工具链** | buf + Kratos 插件 | 生成 pb.go / grpc / http / errors |
| **数据库迁移** | dbmate | 手写 SQL migration，管理 schema 版本 |
| **数据访问** | GORM Gen + rawsql | 从 migration SQL 文件生成 model + 类型安全查询 |
| **依赖注入** | Wire | 编译期依赖注入，零运行时反射 |
| **日志** | slog + Kratos adapter | 统一日志管道 |

**架构风格**：域模块化 —— 按业务域切模块，域内按职责分文件，不按用例拆子目录。

---

## 开发者只需要碰的文件

| 你写什么 | 在哪里写 | 写什么内容 |
|---|---|---|
| **Proto** | `api/user/v1/user.proto` | message + service + http annotation |
| **错误码** | `api/user/v1/error_reason.proto` | 错误枚举，Kratos 生成辅助函数 |
| **Migration SQL** | `db/migrations/*.sql` | dbmate 格式的建表/改表 DDL |
| **UseCase** | `internal/feature/user/usecase.go` | 纯业务逻辑 |
| **跨域接口** | `internal/domain/ports/*.go` | interface 定义 |
| **跨域实现** | `internal/feature/user/facade.go` | 暴露给其他域的能力 |

**其余全部自动生成**：Kratos service 接口、HTTP/gRPC 路由注册、GORM model/query、Wire 胶水、错误辅助函数。

---

## 目录结构

```
project/
├── cmd/server/
│   ├── main.go                      # Kratos app 启动
│   └── wire.go                      # Wire injector
├── api/                             # Proto 定义
│   ├── user/v1/
│   │   ├── user.proto               # message + service + http/grpc
│   │   └── error_reason.proto       # 错误码枚举
│   └── order/v1/
│       ├── order.proto
│       └── error_reason.proto
├── gen/                             # 全部生成代码（gitignore 或提交均可）
│   ├── api/user/v1/
│   │   ├── user.pb.go               # protoc-gen-go
│   │   ├── user_grpc.pb.go          # protoc-gen-go-grpc
│   │   ├── user_http.pb.go          # protoc-gen-go-http（Kratos）
│   │   └── error_reason.pb.go       # protoc-gen-go-errors（Kratos）
│   ├── model/                       # GORM Gen 生成的 struct
│   │   ├── user.gen.go
│   │   └── order.gen.go
│   └── query/                       # GORM Gen 生成的类型安全 DAO
│       ├── user.gen.go
│       └── order.gen.go
├── db/
│   └── migrations/                  # dbmate SQL migration
│       ├── 20240101000000_create_users.sql
│       └── 20240101000001_create_orders.sql
├── internal/
│   ├── domain/
│   │   └── ports/                   # 跨域契约（只有接口）
│   │       ├── user.go              # type UserQuery interface
│   │       └── order.go             # type OrderQuery interface
│   ├── feature/                     # 业务域模块
│   │   ├── user/
│   │   │   ├── domain.go            # 领域实体 + 值对象 + 业务规则
│   │   │   ├── usecase.go           # 所有用例逻辑
│   │   │   ├── repo.go              # repository 接口 + 实现
│   │   │   ├── facade.go            # 对外暴露的能力（实现 ports 接口）
│   │   │   ├── service.go           # Kratos 接口实现（薄胶水）
│   │   │   └── wire.go              # Wire provider
│   │   └── order/
│   │       ├── domain.go
│   │       ├── usecase.go
│   │       ├── repo.go
│   │       ├── facade.go
│   │       ├── service.go
│   │       └── wire.go
│   ├── platform/                    # 基础设施
│   │   ├── database/conn.go         # GORM 连接 + 事务管理
│   │   └── logger/kratos_adapter.go # slog → Kratos log 适配
│   └── shared/
│       └── middleware/              # Kratos middleware
├── cmd/gormgen/
│   └── main.go                      # GORM Gen 配置入口
├── buf.yaml
├── buf.gen.yaml
├── .golangci.yml
└── Makefile
```

---

## 生成流水线

```
make generate
    │
    ├─ Step 1: buf generate
    │    ├─ protoc-gen-go              → pb.go
    │    ├─ protoc-gen-go-grpc         → _grpc.pb.go
    │    ├─ protoc-gen-go-http         → _http.pb.go（Kratos）
    │    ├─ protoc-gen-go-errors       → error_reason.pb.go（Kratos）
    │    └─ protoc-gen-validate        → .pb.validate.go（请求校验）
    │
    ├─ Step 2: gorm gen（通过 rawsql 读取 migration SQL，无需连接数据库）
    │    ├─ gen/model/                 → 表对应的 Go struct
    │    └─ gen/query/                 → 类型安全的查询 API
    │
    ├─ Step 3: 自定义 codegen（可选，非常轻量）
    │    ├─ service.go 胶水（从 proto service 定义推导）
    │    ├─ wire ProviderSet
    │    └─ usecase 骨架（仅文件不存在时创建）
    │
    └─ Step 4: wire generate           → wire_gen.go
```

### 数据访问生成链路

```
dbmate migration SQL 文件
    ↓  rawsql 直接读取（无需运行数据库）
GORM Gen
    ↓
gen/model/   → Go struct（对应数据库表）
gen/query/   → 类型安全的查询方法
```

GORM Gen 配置入口：

```go
// cmd/gormgen/main.go
package main

import (
    "gorm.io/gen"
    "gorm.io/gorm"
    "github.com/go-gorm/rawsql"
)

func main() {
    g := gen.NewGenerator(gen.Config{
        OutPath:      "gen/query",
        ModelPkgPath: "gen/model",
    })

    db, _ := gorm.Open(rawsql.New(rawsql.Config{
        FilePath: []string{"db/migrations"},  // 直接读取 migration SQL 文件
    }))

    g.UseDB(db)

    // 为所有表生成基础 CRUD
    g.ApplyBasic(g.GenerateAllTable()...)

    g.Execute()
}
```

**关键优势**：`rawsql` 让 GORM Gen 可以从 SQL 文件直接解析表结构，CI/CD 环境不需要运行数据库实例就能完成代码生成。

---

## 每个文件的职责与代码示例

### Proto 定义

```protobuf
// api/user/v1/user.proto
syntax = "proto3";
package user.v1;

import "google/api/annotations.proto";
import "validate/validate.proto";

service UserService {
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
        option (google.api.http) = {
            post: "/api/v1/users"
            body: "*"
        };
    }
    rpc GetUser(GetUserRequest) returns (GetUserResponse) {
        option (google.api.http) = {
            get: "/api/v1/users/{id}"
        };
    }
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
        option (google.api.http) = {
            get: "/api/v1/users"
        };
    }
}

message CreateUserRequest {
    string name = 1  [(validate.rules).string = {min_len: 1, max_len: 100}];
    string email = 2 [(validate.rules).string.email = true];
}
message CreateUserResponse {
    int64 id = 1;
    string name = 2;
}

message GetUserRequest {
    int64 id = 1;
}
message GetUserResponse {
    int64 id = 1;
    string name = 2;
    string email = 3;
}

message ListUsersRequest {
    int32 page = 1;
    int32 page_size = 2;
}
message ListUsersResponse {
    repeated GetUserResponse users = 1;
    int64 total = 2;
}
```

```protobuf
// api/user/v1/error_reason.proto
syntax = "proto3";
package user.v1;

import "errors/errors.proto";

enum ErrorReason {
    option (errors.default_code) = 500;

    USER_NOT_FOUND = 0     [(errors.code) = 404];
    USER_ALREADY_EXISTS = 1 [(errors.code) = 409];
    USER_INVALID_EMAIL = 2  [(errors.code) = 400];
}
```

Kratos 的 `protoc-gen-go-errors` 会自动生成辅助函数：

```go
// gen/api/user/v1/error_reason.pb.go [AUTO-GENERATED]
func ErrorUserNotFound(format string, args ...interface{}) *errors.Error { ... }
func IsUserNotFound(err error) bool { ... }
```

### Migration SQL

```sql
-- db/migrations/20240101000000_create_users.sql

-- migrate:up
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS users;
```

### domain.go — 纯业务逻辑，零外部依赖

```go
// internal/feature/order/domain.go
package order

import "time"

// ============ 实体 ============

type Order struct {
    ID        int64
    UserID    int64
    Items     []OrderItem
    Status    Status
    CreatedAt time.Time
}

type OrderItem struct {
    ProductID  int64
    Quantity   int32
    PriceCents int64
}

// ============ 值对象 ============

type Status int

const (
    StatusPending   Status = 1
    StatusPaid      Status = 2
    StatusCancelled Status = 3
)

func (s Status) CanTransitionTo(target Status) bool {
    switch s {
    case StatusPending:
        return target == StatusPaid || target == StatusCancelled
    case StatusPaid:
        return target == StatusCancelled
    default:
        return false
    }
}

// ============ 域服务 ============

func TotalAmount(items []OrderItem) int64 {
    var total int64
    for _, item := range items {
        total += item.PriceCents * int64(item.Quantity)
    }
    return total
}
```

对于简单域（如 user），如果没有复杂业务规则，`domain.go` 可以省略，直接用 proto message + GORM model 两层即可。

### repo.go — 数据访问（具体类型，无接口）

UseCase 直接依赖具体的 `*Repo`，不抽接口。Go 的隐式接口允许事后加接口而不改 repo.go。

```go
// internal/feature/order/repo.go
package order

import (
    "context"
    "project/gen/model"
    "project/gen/query"

    "gorm.io/gorm"
)

type Repo struct {
    db *gorm.DB
    q  *query.Query
}

func NewRepo(db *gorm.DB) *Repo {
    return &Repo{db: db, q: query.Use(db)}
}

func (r *Repo) CreateOrder(ctx context.Context, userID int64, items []OrderItem, total int64) (*model.Order, error) {
    o := &model.Order{UserID: userID, TotalCents: total, Status: int(StatusPending)}
    err := r.q.Order.WithContext(ctx).Create(o)
    return o, err
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
    return r.q.Order.WithContext(ctx).Where(r.q.Order.ID.Eq(id)).First()
}

// WithTx 事务支持——返回一个事务内的 Repo
func (r *Repo) WithTx(fn func(tx *Repo) error) error {
    return r.q.Transaction(func(tx *query.Query) error {
        return fn(&Repo{db: r.db, q: tx})
    })
}
```

如果将来 UseCase 需要 mock Repo 做单测，在 usecase.go 里加一行接口定义即可，repo.go 不用改。

### usecase.go — 所有用例集中在一个文件

```go
// internal/feature/order/usecase.go
package order

import (
    "context"
    "log/slog"

    orderpb "project/gen/api/order/v1"
    "project/internal/domain/ports"
)

type UseCase struct {
    repo      *Repo
    userQuery ports.UserQuery // 跨域依赖
    logger    *slog.Logger
}

func NewUseCase(repo *Repo, uq ports.UserQuery, logger *slog.Logger) *UseCase {
    return &UseCase{repo: repo, userQuery: uq, logger: logger}
}

// ---- 创建订单 ----

func (uc *UseCase) Create(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
    // 跨域调用：确认用户存在
    exists, err := uc.userQuery.ExistsByID(ctx, req.UserId)
    if err != nil {
        return nil, err
    }
    if !exists {
        return nil, orderpb.ErrorUserNotFound("user %d", req.UserId)
    }

    items := convertItemsFromPB(req.Items)
    total := TotalAmount(items) // 调用本域的域逻辑

    // 事务：创建订单 + 订单项
    var order *Order
    err = uc.repo.WithTx(func(tx *Repo) error {
        var txErr error
        order, txErr = tx.CreateOrder(ctx, req.UserId, items, total)
        if txErr != nil {
            return txErr
        }
        return tx.CreateOrderItems(ctx, order.ID, items)
    })
    if err != nil {
        return nil, err
    }

    return &orderpb.CreateOrderResponse{Id: order.ID, Total: total}, nil
}

// ---- 取消订单 ----

func (uc *UseCase) Cancel(ctx context.Context, req *orderpb.CancelOrderRequest) (*orderpb.CancelOrderResponse, error) {
    order, err := uc.repo.GetByID(ctx, req.Id)
    if err != nil {
        return nil, err
    }

    if !order.Status.CanTransitionTo(StatusCancelled) {
        return nil, orderpb.ErrorInvalidStatus("cannot cancel order in status %d", order.Status)
    }

    if err := uc.repo.UpdateStatus(ctx, req.Id, StatusCancelled); err != nil {
        return nil, err
    }

    return &orderpb.CancelOrderResponse{}, nil
}

// ---- 查询列表 ----

func (uc *UseCase) List(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
    orders, total, err := uc.repo.ListByUser(ctx, req.UserId, int(req.Page), int(req.PageSize))
    if err != nil {
        return nil, err
    }
    return convertListResponse(orders, total), nil
}

// ---- 转换函数（不超 300 行就放这里，超了拆 convert.go）----

func convertItemsFromPB(items []*orderpb.OrderItem) []OrderItem {
    result := make([]OrderItem, len(items))
    for i, item := range items {
        result[i] = OrderItem{
            ProductID:  item.ProductId,
            Quantity:   item.Quantity,
            PriceCents: item.PriceCents,
        }
    }
    return result
}

func convertListResponse(orders []*Order, total int64) *orderpb.ListOrdersResponse {
    // ... 转换逻辑 ...
    return &orderpb.ListOrdersResponse{Total: total}
}
```

单文件超 300 行时，按用例拆为 `usecase_create.go`、`usecase_cancel.go`，同包平级，不建子目录。

### service.go — Kratos 接口实现（薄胶水）

Kratos 从 proto 生成接口（在 `user_http.pb.go` 里）：

```go
// gen/api/order/v1/order_http.pb.go [KRATOS 生成]
type OrderServiceHTTPServer interface {
    CreateOrder(context.Context, *CreateOrderRequest) (*CreateOrderResponse, error)
    CancelOrder(context.Context, *CancelOrderRequest) (*CancelOrderResponse, error)
    ListOrders(context.Context, *ListOrdersRequest) (*ListOrdersResponse, error)
}
```

service.go 把这个接口委托给 UseCase：

```go
// internal/feature/order/service.go
package order

import (
    "context"
    orderpb "project/gen/api/order/v1"
)

type Service struct {
    orderpb.UnimplementedOrderServiceServer
    uc *UseCase
}

func NewService(uc *UseCase) *Service {
    return &Service{uc: uc}
}

func (s *Service) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
    return s.uc.Create(ctx, req)
}

func (s *Service) CancelOrder(ctx context.Context, req *orderpb.CancelOrderRequest) (*orderpb.CancelOrderResponse, error) {
    return s.uc.Cancel(ctx, req)
}

func (s *Service) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
    return s.uc.List(ctx, req)
}
```

每个方法一行转发。新增 rpc 时加一个方法即可。**这个文件可以由 codegen 自动生成**。

### facade.go — 对外暴露的跨域能力

```go
// internal/feature/user/facade.go
package user

import "context"

// Facade 实现 ports.UserQuery，暴露给其他域使用
type Facade struct {
    repo Repository
}

func NewFacade(repo Repository) *Facade {
    return &Facade{repo: repo}
}

func (f *Facade) ExistsByID(ctx context.Context, id int64) (bool, error) {
    return f.repo.ExistsByID(ctx, id)
}
```

```go
// internal/domain/ports/user.go
package ports

import "context"

type UserQuery interface {
    ExistsByID(ctx context.Context, id int64) (bool, error)
}
```

跨域调用的完整链路：

```
order.UseCase
    → 依赖 ports.UserQuery（接口）
    → Wire 注入 user.Facade（实现）
    → user.Facade 调用 user.Repository
    → order 模块完全不知道 user 模块的存在
```

### wire.go — Wire provider 声明

```go
// internal/feature/user/wire.go
package user

import (
    "github.com/google/wire"
    "project/internal/domain/ports"
)

var ProviderSet = wire.NewSet(
    NewRepo,
    NewUseCase,
    NewService,
    NewFacade,
    wire.Bind(new(ports.UserQuery), new(*Facade)),
)
```

```go
// internal/feature/order/wire.go
package order

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
    NewRepo,
    NewUseCase,
    NewService,
)
```

### cmd/server — 应用启动

```go
// cmd/server/main.go
package main

import (
    "flag"
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
    configPath := flag.String("conf", "configs/config.yaml", "config path")
    flag.Parse()

    bc := loadConfig(*configPath)
    logger := initLogger()

    app, cleanup, err := wireApp(bc, logger)
    if err != nil {
        panic(err)
    }
    defer cleanup()

    if err := app.Run(); err != nil {
        panic(err)
    }
}

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server) *kratos.App {
    return kratos.New(
        kratos.Logger(logger),
        kratos.Server(hs, gs),
    )
}
```

```go
// cmd/server/wire.go
//go:build wireinject

package main

import (
    "github.com/google/wire"
    "github.com/go-kratos/kratos/v2"

    "project/internal/feature/user"
    "project/internal/feature/order"
    "project/internal/platform"
)

func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(
        platform.ProviderSet,   // DB 连接、logger adapter
        user.ProviderSet,       // user 域全部依赖
        order.ProviderSet,      // order 域全部依赖
        newHTTPServer,          // 注册 service 到 HTTP server
        newGRPCServer,          // 注册 service 到 gRPC server
        newApp,
    ))
}
```

```go
// cmd/server/server.go
package main

import (
    "github.com/go-kratos/kratos/v2/middleware/validate"
    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/go-kratos/kratos/v2/transport/grpc"

    userpb "project/gen/api/user/v1"
    orderpb "project/gen/api/order/v1"
    userfeature "project/internal/feature/user"
    orderfeature "project/internal/feature/order"
)

func newHTTPServer(conf *conf.Server, userSvc *userfeature.Service, orderSvc *orderfeature.Service) *http.Server {
    srv := http.NewServer(
        http.Address(conf.Http.Addr),
        http.Timeout(conf.Http.Timeout.AsDuration()),
        http.Middleware(
            validate.Validator(), // proto validate 自动校验
        ),
    )
    userpb.RegisterUserServiceHTTPServer(srv, userSvc)
    orderpb.RegisterOrderServiceHTTPServer(srv, orderSvc)
    return srv
}

func newGRPCServer(conf *conf.Server, userSvc *userfeature.Service, orderSvc *orderfeature.Service) *grpc.Server {
    srv := grpc.NewServer(
        grpc.Address(conf.Grpc.Addr),
        grpc.Timeout(conf.Grpc.Timeout.AsDuration()),
        grpc.Middleware(
            validate.Validator(),
        ),
    )
    userpb.RegisterUserServiceServer(srv, userSvc)
    orderpb.RegisterOrderServiceServer(srv, orderSvc)
    return srv
}
```

---

## 跨域调用总结

```
┌─────────────────────────────────────────────────┐
│  internal/domain/ports/                         │
│    user.go   → type UserQuery interface         │
│    order.go  → type OrderQuery interface        │
│  （纯接口，不依赖任何 feature 包）               │
└──────────────────────┬──────────────────────────┘
                       │ 实现
        ┌──────────────┴──────────────┐
        ▼                             ▼
  feature/user/                 feature/order/
  facade.go                     facade.go
  (实现 ports.UserQuery)        (实现 ports.OrderQuery)
        │                             │
        │ Wire Bind                   │ Wire Bind
        ▼                             ▼
  order/usecase.go              其他域/usecase.go
  (注入 ports.UserQuery)        (注入 ports.OrderQuery)
```

规则：
- **ports/** 只有接口定义，零 import
- **facade.go** 实现 ports 接口，是域的"对外 API"
- **usecase.go** 只依赖 ports 接口，不 import 其他 feature 包
- **Wire** 负责把 facade 绑定到 ports 接口

---

## 错误处理

使用 Kratos 原生 proto-first 错误体系，不需要自定义 errx 包：

```go
// usecase 中使用
func (uc *UseCase) Get(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
    user, err := uc.repo.GetByID(ctx, req.Id)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, userpb.ErrorUserNotFound("user %d", req.Id)
    }
    return &userpb.GetUserResponse{Id: user.ID, Name: user.Name, Email: user.Email}, nil
}
```

Kratos 自动处理：
- HTTP 返回 `{"code": 404, "reason": "USER_NOT_FOUND", "message": "user 123"}`
- gRPC 映射到对应的 status code
- 错误码、HTTP 状态码、gRPC 状态码全部从 proto 定义走

---

## 日志

Kratos 有自己的 `log.Logger` 接口，写一个 slog adapter 统一：

```go
// internal/platform/logger/kratos_adapter.go
package logger

import (
    "log/slog"
    kratoslog "github.com/go-kratos/kratos/v2/log"
)

type SlogAdapter struct {
    logger *slog.Logger
}

func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
    return &SlogAdapter{logger: logger}
}

func (a *SlogAdapter) Log(level kratoslog.Level, keyvals ...interface{}) error {
    // 转发到 slog
    return nil
}
```

Kratos 内部日志和业务日志走同一个输出管道。

---

## 文件拆分规则

**单文件超 300 行就拆**。拆法是按用例拆为平级文件，不建子目录：

```
feature/order/
├── domain.go
├── usecase_create.go        # 拆出来的
├── usecase_cancel.go        # 拆出来的
├── usecase_list.go          # 拆出来的
├── repo.go
├── facade.go
├── service.go
└── wire.go
```

Go 的同包文件天然共享类型和函数，拆文件不需要改任何 import。

---

## 测试策略

| 测试类型 | 怎么测 | 在哪里 |
|---|---|---|
| **UseCase 单测** | mock Repository 接口 | `feature/order/usecase_test.go` |
| **域逻辑单测** | 纯函数，直接测 | `feature/order/domain_test.go` |
| **Repository 集成测试** | testcontainers + 真实 PG | `feature/order/repo_test.go` |
| **API 集成测试** | Kratos `http.NewClient` | `tests/integration/` |

Repository 定义为接口，UseCase 依赖接口，天然可 mock：

```go
// feature/order/usecase_test.go
func TestCreate_UserNotFound(t *testing.T) {
    mockRepo := &MockRepository{}
    mockUserQuery := &MockUserQuery{existsByID: false}
    uc := NewUseCase(mockRepo, mockUserQuery, slog.Default())

    _, err := uc.Create(context.Background(), &orderpb.CreateOrderRequest{UserId: 999})
    assert.True(t, orderpb.IsUserNotFound(err))
}
```

---

## buf.gen.yaml 配置

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
  - local: protoc-gen-go-http
    out: gen
    opt: paths=source_relative
  - local: protoc-gen-go-errors
    out: gen
    opt: paths=source_relative
  - remote: buf.build/envoyproxy/protoc-gen-validate
    out: gen
    opt: paths=source_relative,lang=go
```

---

## 开发者最终工作流

```
新增一个 "删除用户" 端点:

1. user.proto 加 rpc DeleteUser                              (3 行)
2. error_reason.proto 加错误码（如需要）                       (1 行)
3. db/migrations/ 加 migration（如需 schema 变更）             (可选)
4. make generate                                              (全部胶水自动生成)
5. usecase.go 加 Delete 方法                                  (10-20 行)
6. service.go 加 DeleteUser 转发（或 codegen 自动生成）        (3 行)
7. (跨域才需要) domain/ports/ 加接口 + facade.go 加实现        (偶尔)
```

手写总代码量约 **20 行**，得到完整的 HTTP + gRPC 双协议端点 + 请求校验 + 错误码 + 类型安全查询 + OpenAPI 文档。

---

## 为什么选择域模块化而非垂直切片

| 维度 | 垂直切片（域内拆子目录） | 域模块化（当前方案） |
|---|---|---|
| 新增一个端点 | 建目录 + 3 文件 | 在 usecase.go 加一个方法 |
| 理解一个域 | 打开 4-6 个子目录 | 打开 5 个文件 |
| AI 友好度 | 需要遍历子目录树 | 一个目录全部看完 |
| 域内代码复用 | 需要 `domain/` 子包 | 同包直接调用 |
| 文件数量（10 个端点） | 约 30-40 文件 | 约 8-10 文件 |
| 何时需要拆文件 | 一开始就有很多文件 | 单域超 300 行时按需拆 |
