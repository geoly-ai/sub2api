# Sub2API 管理员 MCP

## 概览

Sub2API 提供一个仅管理员可访问的 MCP HTTP 端点，用于通过 MCP 客户端执行常见用户管理操作。

- 端点：`POST /api/v1/admin/mcp`
- 认证：复用管理员后台认证
  - 推荐：`x-api-key: <SUB2API_ADMIN_API_KEY>`
  - 可选：`Authorization: Bearer <SUB2API_ADMIN_JWT>`
- 协议：JSON-RPC 2.0
- MCP 协议版本：`2025-06-18`

> 该端点拥有批量修改用户的能力。请只在可信 MCP 客户端中配置管理员密钥。

## 客户端配置示例

### Admin API Key

```json
{
  "name": "sub2api-admin",
  "type": "http",
  "url": "https://your-domain.com/api/v1/admin/mcp",
  "headers": {
    "x-api-key": "<SUB2API_ADMIN_API_KEY>"
  }
}
```

### 管理员 JWT

```json
{
  "name": "sub2api-admin",
  "type": "http",
  "url": "https://your-domain.com/api/v1/admin/mcp",
  "headers": {
    "Authorization": "Bearer <SUB2API_ADMIN_JWT>"
  }
}
```

## 支持的 MCP 方法

- `initialize`
- `notifications/initialized`
- `tools/list`
- `tools/call`

## 当前工具清单

### `admin_search_users`

查询用户列表，支持分页、搜索、状态、角色和分组过滤。

参数：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | integer | 否 | `1` | 页码，最小为 `1` |
| `page_size` | integer | 否 | `20` | 每页数量，最大为 `100` |
| `search` | string | 否 | `""` | 搜索邮箱或用户名，最长按 100 个字符处理 |
| `status` | string | 否 | `""` | 用户状态：`active`、`disabled` 或空 |
| `role` | string | 否 | `""` | 用户角色：`user`、`admin` 或空 |
| `group_name` | string | 否 | `""` | 按允许分组名称模糊过滤 |
| `api_key_group_id` | integer | 否 | `0` | 按用户 API Key 绑定分组过滤 |
| `include_subscriptions` | boolean | 否 | 后端默认 | 是否包含订阅信息 |
| `sort_by` | string | 否 | `created_at` | 排序字段 |
| `sort_order` | string | 否 | `desc` | `asc` 或 `desc` |

示例：

```json
{
  "page": 1,
  "page_size": 20,
  "search": "alice@example.com",
  "status": "active",
  "sort_by": "created_at",
  "sort_order": "desc"
}
```

### `admin_batch_create_users`

批量创建普通用户。该工具不会创建管理员账号。

参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `users` | array | 是 | 待创建用户列表，最多 500 个 |

`users[]` 字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 用户邮箱 |
| `password` | string | 是 | 初始密码，至少 6 个字符 |
| `username` | string | 否 | 用户名 |
| `notes` | string | 否 | 管理员备注 |
| `balance` | number | 否 | 初始余额，未传时使用系统默认余额 |
| `concurrency` | integer | 否 | 并发数 |
| `rpm_limit` | integer | 否 | 用户级 RPM 限制，`0` 表示不限制 |
| `allowed_groups` | integer[] | 否 | 允许绑定的专属分组 ID |

示例：

```json
{
  "users": [
    {
      "email": "alice@example.com",
      "password": "secret123",
      "username": "alice",
      "balance": 10,
      "concurrency": 2,
      "allowed_groups": [1, 2]
    }
  ]
}
```

### `admin_batch_add_balance`

批量为用户余额加值。内部复用管理员余额调整逻辑，会写入余额调整记录并触发相关缓存失效。

参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_ids` | integer[] | 是 | 目标用户 ID，最多 500 个 |
| `amount` | number | 是 | 每个用户增加的余额，必须大于 `0` |
| `notes` | string | 否 | 写入余额调整记录的备注 |

示例：

```json
{
  "user_ids": [123, 456],
  "amount": 20,
  "notes": "MCP 批量补偿"
}
```

### `admin_batch_disable_users`

批量禁用用户。内部复用管理员用户更新逻辑；管理员账号会被后端保护，无法通过该工具禁用。

参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `user_ids` | integer[] | 是 | 目标用户 ID，最多 500 个 |
| `notes` | string | 否 | 可选管理员备注 |

示例：

```json
{
  "user_ids": [123, 456],
  "notes": "异常使用，批量禁用"
}
```

## 批量工具返回格式

批量工具会逐条处理并返回每一项的结果；单个用户失败不会阻止同批次其他用户继续执行。

```json
{
  "success_count": 1,
  "failure_count": 1,
  "results": [
    {
      "index": 0,
      "user_id": 123,
      "email": "alice@example.com",
      "success": true,
      "user": {}
    },
    {
      "index": 1,
      "user_id": 456,
      "success": false,
      "error": "user not found"
    }
  ]
}
```

## 幂等性

批量写工具支持可选请求头：

```http
Idempotency-Key: <stable-key>
```

当同一管理员、同一工具、同一参数重复提交相同 `Idempotency-Key` 时，后端会复用已有幂等记录，避免重复执行。建议所有批量写入调用都传入稳定的幂等键。

