# 邻里互助平台（go02-neighbor-help）

一个使用 **纯 Go 标准库** 从 0 到 1 构建的小区邻里互助平台。居民发布求助或提供帮助，系统完成匹配、履约确认后双方互相评价，信用分随评价与履约行为动态调整。系统内置前端页面与文件级数据持久化，可通过 Docker 独立运行。

---

## 一、项目简介

在小区场景中，居民经常遇到“临时需要人帮忙取快递”“周末有空可以帮邻居买菜”“老人就医想有人陪同”这类短时互助需求。传统做法是业主群里喊一声，响应乱、承诺无约束、事后无法评价。本系统把互助过程产品化：

- **居民（resident）**：注册登录后发布求助帖（request）或帮助帖（offer），浏览广场、报名应助、接受报名、确认开始与完成，任务完成后互相评价。
- **管理员（admin）**：审核举报、冻结违规账号、强制关闭帖子、调整信用分、查看全站统计、创建居民账号。
- **核心业务规则**：一次互助只能匹配一名对方；完成后双方各评一次；信用分决定紧急求助、高频发帖与应助资格。

系统使用 Go 1.22 + 标准库（`net/http`、`encoding/json`、`embed`、`sync` 等），**零第三方依赖**，可完全离线构建与运行，适合在受限网络环境下交付。

---

## 二、功能特性

### 2.1 用户与权限

| 角色 | 能力 |
|------|------|
| 管理员（admin） | 用户列表/创建/冻结/解冻，强制关闭帖子，处理举报，调整信用，查看全局统计 |
| 居民（resident） | 注册登录、维护楼栋房号资料、发帖、报名、匹配、履约、互评、留言、收藏、举报、查看通知与个人信用 |
| 未登录访客 | 仅可访问登录/注册页与健康检查，不可查看互助内容 |

- 账号体系：首次启动自动创建种子管理员（默认 `admin / admin123`）；居民可自助注册，管理员也可代建账号。
- 会话：登录成功后颁发内存 Token（Bearer），带过期时间；登出、改密、冻结即失效。
- 口令安全：盐值（salt）+ 多轮迭代 SHA-256 口令哈希（演示级实现，生产建议替换为 bcrypt/argon2）。
- 账号状态：`active`（正常）/ `frozen`（冻结，只读不可发帖应助）/ `banned`（封禁，无法登录）。

### 2.2 互助帖生命周期

帖子类型：

- **求助（request）**：我需要帮助，例如“明天下午帮我照看一下猫”。
- **帮助（offer）**：我能提供帮助，例如“周末可以帮忙代买菜”。

状态流转：

```
        创建                 发布                  接受报名
[ 草稿 draft ] ───────► [ 开放 open ] ───────► [ 已匹配 matched ]
        │                       │                      │
        │                       │ 过期 expire          │ 双方确认开始
        │                       │ 取消 cancel          ▼
        │                       ▼               [ 进行中 in_progress ]
        │                 [ 已取消 / 已过期 ]          │
        │                       ▲                      │ 帮助方标记完成
        │                       │                      ▼
        └───────────────────────┴────────────── [ 待确认 pending_confirm ]
                                                       │
                                                       │ 求助方确认 / 超时规则
                                                       ▼
                                                [ 已完成 completed ]
```

- **草稿（draft）**：仅作者与管理员可见。
- **开放（open）**：广场可见，允许报名。
- **已匹配（matched）**：作者接受一名报名者，生成任务，其余待处理报名自动拒绝。
- **进行中（in_progress）**：双方确认开始（或任一方在匹配后确认开始，另一方随后确认）。
- **待确认（pending_confirm）**：帮助方标记完成，等待求助方确认。
- **已完成（completed）**：求助方确认完成，双方可互评。
- **已取消（cancelled）**：作者或管理员关闭；匹配后取消需填写原因，可能扣信用。
- **已过期（expired）**：超过服务时间窗口且仍为开放状态，系统在读取/列表时惰性过期。

附加属性：

- **紧急程度（urgency）**：`low` / `normal` / `high` / `urgent`。紧急帖要求信用分 ≥ 50，受限信用用户不可发紧急帖。
- **分类（category）**：买菜代购、代取快递、照看宠物、搬运重物、上门修理、陪同就医、临时看护、其他。
- **时间窗口**：`TimeWindowStart` / `TimeWindowEnd`，过期后开放帖自动变为 expired。
- **地点**：楼栋 / 单元 / 房号或“小区公共区域”。
- **答谢方式**：无 / 口头感谢 / 积分心意 / 实物答谢（仅记录说明，不涉及资金清算）。

### 2.3 报名与匹配（核心）

> **定义**：一条开放帖在任意时刻最多绑定 **一名** 对方。接受报名即锁定匹配，生成一条任务（Task）。

派生规则：

1. **不能给自己报名**；同一用户对同一帖只能有一条未撤回报名。
2. **求助帖**：报名者是“帮助方（helper）”；**帮助帖**：报名者是“受助方（requester）”。任务上始终同时记录 requester 与 helper。
3. **信用门槛**：信用等级 `restricted`（<40）不能报名紧急/高紧急帖，也不能同时拥有超过 1 条进行中任务。
4. **接受报名**：作者（或管理员代操作）接受后，帖子进入 matched，其余 pending 报名自动 rejected。
5. **拒绝 / 撤回**：作者可拒绝；报名人可在接受前撤回。
6. **开放帖数量上限**：每位居民同时处于 draft/open/matched/in_progress/pending_confirm 的帖子不超过 10 条。

### 2.4 任务履约

任务状态：

| 状态 | 含义 |
|------|------|
| pending_start | 已匹配，等待开始确认 |
| in_progress | 已开始 |
| pending_confirm | 帮助方已标记完成，等待求助方确认 |
| completed | 双方确认完成 |
| cancelled | 取消 |
| disputed | 求助方拒绝确认，进入争议（管理员介入） |

确认规则：

- 匹配后，求助方与帮助方均可“确认开始”；**双方都确认**后进入 in_progress。为避免一方失联卡死，若一方已确认开始超过约定等待策略，管理员可强制推进。
- 仅 **helper** 可“标记完成”，进入 pending_confirm。
- 仅 **requester** 可“确认完成”，进入 completed。
- requester 也可“拒绝确认”并填写原因，任务进入 disputed，通知管理员。
- 开始前取消不扣信用；开始后取消扣发起方 3 分，并关闭帖子。

### 2.5 互评规则（核心）

> **定义**：任务 completed 后，双方可各提交 **一次** 评价。评价维度为 1–5 星 + 标签 + 文字。

1. 未完成任务不可评价；每对（TaskID, FromUserID）唯一。
2. 不能评自己；只能评本次任务的对方。
3. 星级与标签约束：4–5 星只能带正向标签；1–2 星只能带负向标签；3 星可带中性标签。
4. 评价提交后立即按规则调整被评人信用分，并写入信用流水。
5. 评价一旦提交不可修改（管理员可隐藏不当评价，但不改信用流水历史）。
6. 双方都评完后，帖子保持 completed，任务标记 `BothReviewed=true`。

正向标签：守时、热心、靠谱、沟通顺畅、超出预期。  
负向标签：迟到、态度差、未完成、沟通不畅、夸大能力。  
中性标签：一般、有改进空间。

### 2.6 信用分

- 初始分 **60**，范围 **0–100**。
- 等级：`restricted`（<40）/ `new`（40–59）/ `normal`（60–74）/ `trusted`（75–89）/ `excellent`（90–100）。

计分（可叠加，每次变更写 CreditLog）：

| 事件 | 分值 |
|------|------|
| 作为帮助方完成任务 | +2 |
| 作为求助方完成任务 | +1 |
| 收到 5 星 | +3 |
| 收到 4 星 | +1 |
| 收到 3 星 | 0 |
| 收到 2 星 | -2 |
| 收到 1 星 | -5 |
| 开始后主动取消 | -3 |
| 举报成立 | -10 |
| 管理员手工调整 | 自定义（-20～+20） |

受限用户：不可发紧急帖、不可报名高紧急帖、同时进行中任务上限为 1。信用可通过正常履约与好评恢复。

### 2.7 留言、收藏、通知、举报

- **留言**：开放帖允许居民留言与回复；作者可删除自己的留言；管理员可删除任意留言。
- **收藏**：居民可收藏开放帖，便于回访。
- **站内通知**：报名、接受、拒绝、开始、完成、评价、举报处理等事件写入通知箱；打开即标记已读。
- **举报**：可举报用户 / 帖子 / 评价 / 留言。管理员受理后可驳回或成立（成立则视情况冻结账号、关闭帖子、扣信用）。

### 2.8 列表排序与筛选

广场列表默认排序：

1. 紧急程度降序（urgent > high > normal > low）；
2. 信用等级加权（作者 excellent/trusted 略优先）；
3. 发布时间倒序。

支持按类型（求助/帮助）、分类、紧急程度、楼栋、关键词、状态筛选。个人中心可查看“我发布的 / 我报名的 / 我的任务 / 待评价 / 我的收藏 / 通知”。

### 2.9 统计

- **管理员全局**：用户数、活跃/冻结数、帖子各状态数量、任务完成率、平均评分、今日新增互助、待处理举报。
- **个人**：信用分与等级、完成次数、作为帮助/求助次数、平均得分、待办（待开始 / 待确认 / 待评价 / 未读通知）。

### 2.10 数据持久化

- **文件级 JSON 持久化**：用户、帖子、报名、任务、评价、留言、通知、举报、收藏、信用流水保存在单一数据文件（默认 `data/store.json`）。
- 写操作原子落盘（先写临时文件再 `os.Rename`），避免写入中途崩溃损坏数据。
- 服务重启后自动加载恢复全部状态。
- Docker 部署时通过 volume 挂载 `./data` 实现跨容器生命周期持久化。

### 2.11 前端页面

内置轻量前端（HTML + 原生 CSS + 原生 JavaScript，无构建工具，无前端框架），通过 Go `embed` 打包进二进制：

- `/login` 登录与注册
- `/app` 互助广场（发帖、筛选、报名）
- `/posts/{id}` 帖子详情（留言、报名、履约操作）
- `/me` 个人中心（资料、任务、评价、通知、信用流水）
- `/admin` 管理员后台（用户、举报、统计、强制关闭）
- `/static/*` 静态资源

---

## 三、业务逻辑详述

### 3.1 可见性与权限矩阵

| 操作 | 管理员 | 居民（作者） | 居民（对方/他人） | 未登录 |
|------|--------|--------------|-------------------|--------|
| 查看草稿 | ✅ | ✅ | ❌ 404 | ❌ 401 |
| 查看开放帖 | ✅ | ✅ | ✅ | ❌ 401 |
| 发帖 / 编辑草稿 | ✅ | ✅ | ❌ 403 | ❌ 401 |
| 报名 | ✅（一般不使用） | ❌ 不能报自己 | ✅ | ❌ 401 |
| 接受/拒绝报名 | ✅ | ✅ | ❌ 403 | ❌ 401 |
| 确认开始 / 完成 | ✅ | 按角色 | 按角色 | ❌ 401 |
| 互评 | ✅（只读） | 完成后评对方 | 完成后评对方 | ❌ 401 |
| 处理举报 / 冻结用户 | ✅ | ❌ | ❌ | ❌ 401 |

冻结用户：可登录查看，不可发帖、报名、留言。封禁用户：无法登录。

### 3.2 匹配与任务推导

```
accept(application) =
    post.Status == open
    AND application.Status == pending
    AND application.PostID == post.ID
    AND 推导 requester/helper
    → 创建 Task(pending_start)
    → post.Status = matched, post.MatchedUserID = 对方, post.TaskID = task.ID
    → 其余 pending 报名 → rejected
    → 通知双方
```

求助帖：`requester = 作者`，`helper = 报名人`。  
帮助帖：`helper = 作者`，`requester = 报名人`。

### 3.3 已完成与互评

```
canReview(user, task) =
    task.Status == completed
    AND user 是 requester 或 helper
    AND 不存在 Review(task.ID, user.ID)
```

信用更新在评价写入成功后同步发生，避免“评了但没加分”的中间态。完成任务的 +1/+2 在任务进入 completed 时发放，与评价加分独立。

### 3.4 过期惰性处理

不单独跑全表扫描定时任务也能保证语义正确：

- `ListPosts` / `GetPost` 时若 `now > TimeWindowEnd` 且状态为 open，则将帖子置为 expired，并把 pending 报名置为 expired。
- 服务启动后可选地做一次扫描（种子之后），与惰性处理叠加，保证广场尽快干净。

### 3.5 输入校验

- 用户名：3–32 字符，字母数字下划线。
- 口令：6–64 字符。
- 显示名：1–32 字符。
- 帖子标题：1–80 字符；正文：1–4000 字符。
- 报名留言 / 评价文字：0–500 字符。
- 星级：1–5 整数。
- 楼栋 / 单元 / 房号：各最长 16 字符。
- 所有写接口对必填字段、长度、枚举、状态机进行校验，失败返回 `400` 与明确错误信息。

### 3.6 并发安全

- 存储层使用 `sync.RWMutex` 保护内存数据结构，读操作并发、写操作独占。
- **接受报名**、**确认完成** 等状态变更在同一把写锁的业务事务语义下完成（service 调用连续 store 写；MemoryStore 每个方法自带锁，通过“先读校验再写、写时再次检查状态”避免双开匹配）。
- 持久化在写后钩子触发（快照落盘），Save 使用独立 `saveMu` 串行化磁盘写。
- 会话管理器内部使用独立的 `sync.RWMutex`。
- HTTP 服务由标准库自动并发处理请求。

### 3.7 优雅关闭

服务收到 `SIGINT` / `SIGTERM` 时：

1. 停止接收新连接；
2. 等待进行中的请求完成（带超时）；
3. 最后一次落盘持久化数据；
4. 退出。

---

## 四、系统架构

### 4.1 分层架构

```
┌──────────────────────────────────────────────────────┐
│  前端页面（embed：HTML/CSS/JS）                          │
└──────────────────────────────────────────────────────┘
                          ▲ JSON / HTML
┌──────────────────────────────────────────────────────┐
│  handler 层    HTTP 路由 + 请求/响应序列化 + 校验入口     │
│  middleware 层 鉴权、角色控制、日志、Recover、CORS        │
│  service 层    业务逻辑（auth/user/post/match/task/      │
│                review/credit/notify/report/stats）      │
│  store 层      数据访问接口 + 内存实现 + 文件持久化        │
│  model 层      领域模型 + DTO + 领域错误                 │
│  auth 层       口令哈希 + 会话管理                       │
└──────────────────────────────────────────────────────┘
                          ▲
┌──────────────────────────────────────────────────────┐
│  data/store.json（文件持久化）                          │
└──────────────────────────────────────────────────────┘
```

### 4.2 目录结构

```
go02-neighbor-help/
├── README.md                  # 本文档
├── BENZHI_README.md           # 项目说明 + 构建/运行/测试命令
├── go.mod
├── main.go                    # 装配依赖、启动服务、种子数据、优雅关闭
├── Dockerfile                 # 运行镜像（单阶段 golang:1.22）
├── docker-compose.yml         # 一键起服务（:8080）
├── .dockerignore
├── benzhi.Dockerfile          # 质检镜像（勿改）
├── build_benzhi_docker.sh     # 质检构建脚本（勿改）
├── go-test.sh                 # 质检测试脚本（可改）
├── go-run.sh                  # 运行脚本（勿改）
├── internal/
│   ├── model/                 # 领域模型、DTO、错误
│   ├── store/                 # Store 接口、内存实现、文件持久化、种子数据
│   ├── auth/                  # 口令哈希、会话管理
│   ├── service/               # 业务逻辑
│   ├── middleware/            # 鉴权、角色、日志、Recover、CORS
│   ├── handler/               # HTTP 处理器
│   ├── server/                # HTTP 服务器、路由装配、优雅关闭
│   ├── config/                # 配置（环境变量）
│   ├── respond/               # JSON 响应
│   ├── validate/              # 字段校验
│   └── web/                   # embed 前端
└── *_test.go / internal/**/*_test.go
```

---

## 五、数据模型

### 5.1 实体摘要

**User**：ID、用户名、口令哈希/盐/迭代、角色、显示名、楼栋单元房号、简介、账号状态、信用分、统计计数、时间戳。

**HelpPost**：ID、类型（request/offer）、状态、分类、紧急程度、标题正文、地点、时间窗口、答谢方式、作者、匹配对象、任务 ID、浏览/报名计数、关闭原因、时间戳。

**Application**：ID、帖子 ID、报名人、留言、状态（pending/accepted/rejected/withdrawn/expired）、决定时间。

**Task**：ID、帖子 ID、requester、helper、状态、双方开始确认、完成时间、取消/争议原因、双方是否已评。

**Review**：ID、任务/帖子、评价人/被评人、以何种角色评价、星级、标签、文字、是否匿名、是否隐藏。

**Message**：ID、帖子、作者、正文、父留言 ID。

**Notification**：ID、接收人、类型、标题、正文、关联对象、已读。

**Report**：ID、举报人、目标类型/ID、原因、状态、处理人、处理说明。

**Favorite**：用户 ID + 帖子 ID。

**CreditLog**：用户 ID、分差、原因、关联 ID、变更后分数。

### 5.2 关系

- User 1—N HelpPost（作者）。
- HelpPost 1—N Application；HelpPost 1—0..1 Task。
- Task 1—0..2 Review（双方各一条）。
- 删除用户时：级联清理其报名、收藏、通知；进行中任务不允许直接删用户（须先冻结）。
- 关闭/过期帖子时：pending 报名一并结束。

---

## 六、API 参考

所有 `/api/**` 接口返回 JSON，错误格式：`{"code":"...","message":"..."}`。需鉴权接口要求请求头 `Authorization: Bearer <token>`。

### 6.1 鉴权与用户

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| POST | `/api/auth/register` | 公开 | 居民自助注册 |
| POST | `/api/auth/login` | 公开 | 登录，返回 token 与用户信息 |
| POST | `/api/auth/logout` | 已登录 | 登出 |
| GET  | `/api/auth/me` | 已登录 | 当前用户（含信用与待办摘要） |
| PUT  | `/api/me/profile` | 已登录 | 更新显示名/楼栋/简介 |
| PUT  | `/api/me/password` | 已登录 | 修改口令并失效其他会话 |
| GET  | `/api/users` | admin | 用户列表 |
| POST | `/api/users` | admin | 创建居民 |
| POST | `/api/users/{id}/freeze` | admin | 冻结 |
| POST | `/api/users/{id}/unfreeze` | admin | 解冻 |
| POST | `/api/users/{id}/credit` | admin | 手工调信用 |
| GET  | `/api/users/{id}` | 已登录 | 公开资料（不含口令） |
| GET  | `/api/users/{id}/reviews` | 已登录 | 某用户收到的评价 |

### 6.2 互助帖

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET | `/api/posts` | 已登录 | 广场列表（默认仅 open；admin 可看全部） |
| GET | `/api/posts/{id}` | 已登录 | 详情（草稿仅作者/管理员） |
| POST | `/api/posts` | 居民 | 创建（默认 draft，可带 publish=true） |
| PUT | `/api/posts/{id}` | 作者/admin | 更新草稿或有限更新开放帖 |
| POST | `/api/posts/{id}/publish` | 作者/admin | 发布为 open |
| POST | `/api/posts/{id}/cancel` | 作者/admin | 取消 |
| GET | `/api/me/posts` | 已登录 | 我发布的 |

### 6.3 报名与匹配

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| POST | `/api/posts/{id}/apply` | 居民 | 报名 |
| DELETE | `/api/applications/{id}` | 报名人 | 撤回 |
| POST | `/api/applications/{id}/accept` | 作者/admin | 接受并生成任务 |
| POST | `/api/applications/{id}/reject` | 作者/admin | 拒绝 |
| GET | `/api/posts/{id}/applications` | 作者/admin | 报名列表 |
| GET | `/api/me/applications` | 已登录 | 我的报名 |

### 6.4 任务与互评

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET | `/api/tasks/{id}` | 当事人/admin | 任务详情 |
| GET | `/api/me/tasks` | 已登录 | 我的任务 |
| POST | `/api/tasks/{id}/confirm-start` | 当事人 | 确认开始 |
| POST | `/api/tasks/{id}/complete` | helper | 标记完成 |
| POST | `/api/tasks/{id}/confirm-complete` | requester | 确认完成 |
| POST | `/api/tasks/{id}/dispute` | requester | 拒绝确认，进入争议 |
| POST | `/api/tasks/{id}/cancel` | 当事人/admin | 取消任务 |
| POST | `/api/tasks/{id}/reviews` | 当事人 | 提交评价 |
| GET | `/api/tasks/{id}/reviews` | 当事人/admin | 查看该任务评价 |

### 6.5 留言 / 收藏 / 通知 / 举报 / 统计

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET/POST | `/api/posts/{id}/messages` | 已登录 | 留言列表 / 发表 |
| DELETE | `/api/messages/{id}` | 作者/admin | 删除留言 |
| POST/DELETE | `/api/posts/{id}/favorite` | 居民 | 收藏 / 取消 |
| GET | `/api/me/favorites` | 已登录 | 我的收藏 |
| GET | `/api/me/notifications` | 已登录 | 通知列表 |
| POST | `/api/me/notifications/{id}/read` | 已登录 | 标记已读 |
| POST | `/api/me/notifications/read-all` | 已登录 | 全部已读 |
| POST | `/api/reports` | 已登录 | 提交举报 |
| GET | `/api/reports` | admin | 举报表 |
| POST | `/api/reports/{id}/handle` | admin | 处理举报 |
| GET | `/api/stats` | admin | 全局统计 |
| GET | `/api/me/credit-logs` | 已登录 | 我的信用流水 |
| GET | `/api/categories` | 已登录 | 分类字典 |

### 6.6 页面与健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 重定向到 `/login` |
| GET | `/login` | 登录注册页 |
| GET | `/app` | 互助广场 |
| GET | `/posts/{id}` | 帖子详情页 |
| GET | `/me` | 个人中心 |
| GET | `/admin` | 管理员后台 |
| GET | `/static/*` | 静态资源 |
| GET | `/healthz` | 健康检查 |

---

## 七、配置

通过环境变量配置（均有默认值）：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `APP_ADDR` | `:8080` | 监听地址 |
| `APP_DATA_PATH` | `data/store.json` | 数据持久化文件路径 |
| `APP_SESSION_TTL` | `24h` | 会话有效期 |
| `APP_SEED_ADMIN` | `true` | 启动时若不存在管理员则创建 |
| `APP_ADMIN_USERNAME` | `admin` | 种子管理员用户名 |
| `APP_ADMIN_PASSWORD` | `admin123` | 种子管理员口令 |
| `APP_SEED_DEMO` | `true` | 启动时若无居民则写入演示居民与示例帖 |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | 优雅关闭超时 |
| `APP_MAX_OPEN_POSTS` | `10` | 每用户同时进行中帖子上限 |

---

## 八、安全说明

- 口令哈希采用盐值 + 多轮迭代 SHA-256（演示级，对抗彩虹表；生产环境建议替换为 bcrypt / argon2）。
- 会话 Token 使用 `crypto/rand` 生成的高熵随机串。
- 不存储明文口令；登录时仅比对哈希。
- 前端 Token 存放于 `localStorage`，请求时通过 `Authorization` 头携带。
- 本系统为社区内部演示用途，未实现完整 HTTPS / CSRF / 限流，生产部署请置于反向代理之后并补充相应防护。

---

## 九、License

本项目用于教学与内部交付，无特定开源协议。
