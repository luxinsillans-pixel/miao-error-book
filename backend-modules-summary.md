# Memos v0.26.1 后端功能模块汇总

## 项目概述
基于 memos v0.26.1 构建的错题本平台后端，采用 Go + MariaDB 技术栈，提供完整的笔记管理、用户系统、附件存储等功能。

## 核心功能模块

### 1. 用户管理模块 (UserService)
**功能点：**
- 用户 CRUD：创建、读取、更新、删除用户
- 用户查询：列表查询、ID/用户名查询
- 用户统计：个人统计、全局用户统计
- 用户设置：通用设置、Webhook 设置管理
- 个人访问令牌 (PAT)：创建、列表、删除长生命周期 API 令牌
- Webhook 管理：用户级 Webhook 的 CRUD
- 通知管理：用户通知的列表、更新、删除

**关键接口：**
- `ListUsers`, `GetUser`, `CreateUser`, `UpdateUser`, `DeleteUser`
- `GetUserStats`, `ListAllUserStats`
- `GetUserSetting`, `UpdateUserSetting`, `ListUserSettings`
- `ListPersonalAccessTokens`, `CreatePersonalAccessToken`, `DeletePersonalAccessToken`
- `ListUserWebhooks`, `CreateUserWebhook`, `UpdateUserWebhook`, `DeleteUserWebhook`
- `ListUserNotifications`, `UpdateUserNotification`, `DeleteUserNotification`

### 2. Memo（错题）管理模块 (MemoService)
**功能点：**
- Memo CRUD：创建、读取、更新、删除错题/笔记
- 列表查询：分页、筛选（标签、内容、状态）、排序（置顶、时间）
- 可见性控制：公开、保护、私有三种权限
- 附件关联：设置和获取 Memo 的附件
- 关系管理：设置和获取 Memo 之间的关系（引用、评论）
- 评论系统：创建评论、获取评论列表
- 反应系统：点赞/表情反应的增删查

**关键接口：**
- `CreateMemo`, `ListMemos`, `GetMemo`, `UpdateMemo`, `DeleteMemo`
- `SetMemoAttachments`, `ListMemoAttachments`
- `SetMemoRelations`, `ListMemoRelations`
- `CreateMemoComment`, `ListMemoComments`
- `ListMemoReactions`, `UpsertMemoReaction`, `DeleteMemoReaction`

### 3. 附件管理模块 (AttachmentService)
**功能点：**
- 附件上传：支持多种文件类型
- 附件列表：分页查询、按 Memo 筛选
- 附件删除：清理存储资源
- EXIF 信息提取：图片元数据解析

**关键接口：**
- `CreateAttachment`, `ListAttachments`, `DeleteAttachment`

### 4. 活动时间线模块 (ActivityService)
**功能点：**
- 活动记录：用户操作日志（创建 memo、评论等）
- 活动查询：按类型、用户筛选

**关键接口：**
- `ListActivities`

### 5. 实例设置模块 (InstanceService)
**功能点：**
- 系统设置：获取和更新实例级别配置
- Memo 相关设置：内容长度限制、可见性限制等
- 实例信息：获取系统版本、运行状态

**关键接口：**
- `GetInstance`, `UpdateInstance`
- `GetInstanceMemoRelatedSetting`, `UpdateInstanceMemoRelatedSetting`

### 6. 身份提供者模块 (IdpService)
**功能点：**
- 身份提供者管理：OAuth/SSO 配置
- 第三方登录：支持 GitHub、Google 等

**关键接口：**
- `ListIdps`, `CreateIdp`, `UpdateIdp`, `DeleteIdp`

### 7. 快捷方式模块 (ShortcutService)
**功能点：**
- 快捷方式管理：常用查询的保存和调用
- 筛选器语法：CEL 表达式过滤 Memo

**关键接口：**
- `ListShortcuts`, `GetShortcut`, `CreateShortcut`, `UpdateShortcut`, `DeleteShortcut`

### 8. 认证授权模块 (AuthService)
**功能点：**
- 用户登录：用户名/密码认证
- 会话管理：JWT 令牌颁发和验证
- 客户端信息：获取当前客户端上下文

**关键接口：**
- `Login`, `Logout`, `GetAuthStatus`

## 数据库模型
主要数据表：
- `user`：用户基本信息
- `memo`：错题/笔记核心内容
- `attachment`：附件存储信息
- `memo_relation`：Memo 间关系（评论、引用）
- `reaction`：用户对 Memo 的反应
- `activity`：用户活动日志
- `inbox`：用户收件箱（通知）
- `personal_access_token`：长期访问令牌
- `user_setting`：用户个性化配置
- `instance_setting`：系统全局配置
- `idp`：身份提供者配置
- `shortcut`：保存的查询快捷方式

## 扩展点（错题本定制）
基于现有模块可扩展的功能：

### 1. 错题专属字段
- 科目分类：语文、数学、英语、科学等
- 章节/知识点：关联教材章节
- 错因分析：粗心、概念不清、方法错误等
- 难度星级：1-5 星难度评级
- 掌握程度：未掌握、一般、熟练掌握
- 复习计划：下次复习时间、复习次数

### 2. 统计分析功能
- 错题分布：按科目、章节统计
- 进步趋势：时间维度掌握程度变化
- 薄弱环节：高频错题知识点识别
- 班级对比：同班级学生错题对比

### 3. 协作功能
- 老师批注：教师添加点评和建议
- 同学互助：错题分享和讨论
- 班级错题集：公共错题池

### 4. 复习系统
- 艾宾浩斯遗忘曲线提醒
- 自动生成复习计划
- 错题重做和验证

## 技术特性
- **API 设计**：遵循 Google API 设计规范（AIP）
- **认证方式**：JWT + 个人访问令牌
- **数据库支持**：SQLite（开发）/ MySQL·MariaDB（生产）
- **文件存储**：本地存储或 S3 兼容存储
- **国际化**：多语言支持
- **扩展性**：插件系统和 Webhook 支持

## 部署架构
- 容器化：Docker + Docker Compose
- 数据库：MariaDB 11.6
- 前端服务：Nginx 静态文件服务
- 后端服务：Go 二进制服务
- 存储卷：数据持久化配置

---
*文档生成时间：2026-02-10*
*基于 memos v0.26.1 代码分析*