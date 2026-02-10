# Memos v0.26.1 后端测试用例

## 测试环境
- 后端服务：http://localhost:5230
- 认证方式：Bearer Token (JWT)
- 测试工具：curl / Postman / 自动化测试脚本

## 1. 用户管理模块测试

### 1.1 用户认证测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-AUTH-001 | 有效用户登录 | POST | `/api/v1/auth/login` | 返回 200 OK 和 JWT token |
| TC-AUTH-002 | 无效密码登录 | POST | `/api/v1/auth/login` | 返回 401 Unauthorized |
| TC-AUTH-003 | 获取当前用户状态 | GET | `/api/v1/auth/status` | 返回 200 OK 和用户信息 |

### 1.2 用户 CRUD 测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-USER-001 | 创建新用户 | POST | `/api/v1/users` | 返回 201 Created |
| TC-USER-002 | 获取用户列表 | GET | `/api/v1/users` | 返回 200 OK 和用户数组 |
| TC-USER-003 | 按 ID 获取用户 | GET | `/api/v1/users/{id}` | 返回 200 OK 和用户详情 |
| TC-USER-004 | 按用户名获取用户 | GET | `/api/v1/users/{username}` | 返回 200 OK 和用户详情 |
| TC-USER-005 | 更新用户信息 | PATCH | `/api/v1/users/{id}` | 返回 200 OK 和更新后数据 |
| TC-USER-006 | 删除用户 | DELETE | `/api/v1/users/{id}` | 返回 204 No Content |

### 1.3 用户统计测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-STATS-001 | 获取用户统计 | GET | `/api/v1/users/{id}:getStats` | 返回 200 OK 和统计信息 |
| TC-STATS-002 | 获取所有用户统计 | GET | `/api/v1/users:stats` | 返回 200 OK 和统计列表 |

## 2. Memo（错题）管理模块测试

### 2.1 Memo CRUD 测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-MEMO-001 | 创建公开 Memo | POST | `/api/v1/memos` | 返回 201 Created |
| TC-MEMO-002 | 创建私有 Memo | POST | `/api/v1/memos` | 返回 201 Created |
| TC-MEMO-003 | 获取 Memo 列表 | GET | `/api/v1/memos` | 返回 200 OK 和分页数据 |
| TC-MEMO-004 | 按 ID 获取 Memo | GET | `/api/v1/memos/{id}` | 返回 200 OK 和 Memo 详情 |
| TC-MEMO-005 | 更新 Memo 内容 | PATCH | `/api/v1/memos/{id}` | 返回 200 OK 和更新后数据 |
| TC-MEMO-006 | 删除 Memo | DELETE | `/api/v1/memos/{id}` | 返回 204 No Content |

### 2.2 Memo 筛选排序测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-FILTER-001 | 按标签筛选 | GET | `/api/v1/memos?filter=tag:"数学"` | 返回包含该标签的 Memo |
| TC-FILTER-002 | 按内容搜索 | GET | `/api/v1/memos?filter=content:"函数"` | 返回内容包含关键词的 Memo |
| TC-FILTER-003 | 按时间排序 | GET | `/api/v1/memos?order_by=display_time desc` | 返回按时间降序排列 |
| TC-FILTER-004 | 置顶优先排序 | GET | `/api/v1/memos?order_by=pinned desc` | 返回置顶 Memo 在前 |

### 2.3 Memo 附件测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-ATTACH-001 | 为 Memo 添加附件 | PATCH | `/api/v1/memos/{id}/attachments` | 返回 200 OK |
| TC-ATTACH-002 | 获取 Memo 附件列表 | GET | `/api/v1/memos/{id}/attachments` | 返回附件数组 |
| TC-ATTACH-003 | 删除附件 | DELETE | `/api/v1/attachments/{id}` | 返回 204 No Content |

### 2.4 Memo 评论测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-COMMENT-001 | 为 Memo 添加评论 | POST | `/api/v1/memos/{id}/comments` | 返回 201 Created |
| TC-COMMENT-002 | 获取 Memo 评论列表 | GET | `/api/v1/memos/{id}/comments` | 返回评论数组 |
| TC-COMMENT-003 | 嵌套评论支持 | POST | `/api/v1/memos/{id}/comments` | 支持多级评论 |

### 2.5 Memo 反应测试
| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-REACT-001 | 添加表情反应 | POST | `/api/v1/memos/{id}/reactions` | 返回 200 OK 和反应信息 |
| TC-REACT-002 | 获取反应列表 | GET | `/api/v1/memos/{id}/reactions` | 返回反应数组 |
| TC-REACT-003 | 删除反应 | DELETE | `/api/v1/memos/{id}/reactions/{reactionId}` | 返回 204 No Content |

## 3. 附件管理模块测试

| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-FILE-001 | 上传图片附件 | POST | `/api/v1/attachments` | 返回 201 Created 和附件信息 |
| TC-FILE-002 | 上传文档附件 | POST | `/api/v1/attachments` | 支持 PDF、Word 等格式 |
| TC-FILE-003 | 获取附件列表 | GET | `/api/v1/attachments` | 返回分页附件列表 |
| TC-FILE-004 | 删除附件 | DELETE | `/api/v1/attachments/{id}` | 返回 204 No Content |

## 4. 实例设置模块测试

| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-INST-001 | 获取实例信息 | GET | `/api/v1/instance` | 返回系统版本和状态 |
| TC-INST-002 | 更新实例设置 | PATCH | `/api/v1/instance` | 返回 200 OK 和更新后设置 |
| TC-INST-003 | 获取 Memo 相关设置 | GET | `/api/v1/instance/memoRelatedSetting` | 返回内容长度限制等配置 |

## 5. 身份提供者模块测试

| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-IDP-001 | 获取 IDP 列表 | GET | `/api/v1/idps` | 返回身份提供者列表 |
| TC-IDP-002 | 创建 GitHub IDP | POST | `/api/v1/idps` | 返回 201 Created |
| TC-IDP-003 | 更新 IDP 配置 | PATCH | `/api/v1/idps/{id}` | 返回 200 OK |
| TC-IDP-004 | 删除 IDP | DELETE | `/api/v1/idps/{id}` | 返回 204 No Content |

## 6. 快捷方式模块测试

| 测试用例 ID | 测试场景 | 请求方法 | 端点 | 预期结果 |
|------------|----------|----------|------|----------|
| TC-SHORT-001 | 创建快捷方式 | POST | `/api/v1/shortcuts` | 返回 201 Created |
| TC-SHORT-002 | 获取快捷方式列表 | GET | `/api/v1/shortcuts` | 返回用户快捷方式 |
| TC-SHORT-003 | 更新快捷方式 | PATCH | `/api/v1/shortcuts/{id}` | 返回 200 OK |
| TC-SHORT-004 | 删除快捷方式 | DELETE | `/api/v1/shortcuts/{id}` | 返回 204 No Content |

## 7. 集成测试用例

### 7.1 错题本业务流程测试
| 测试用例 ID | 测试场景 | 步骤 | 预期结果 |
|------------|----------|------|----------|
| TC-FLOW-001 | 完整错题录入流程 | 1. 登录<br>2. 创建错题 Memo<br>3. 添加附件（题目图片）<br>4. 添加标签"数学"<br>5. 设置难度星级 | 错题完整保存并可检索 |
| TC-FLOW-002 | 错题复习流程 | 1. 筛选"未掌握"错题<br>2. 按章节分组<br>3. 批量更新掌握程度 | 错题状态正确更新 |
| TC-FLOW-003 | 教师批注流程 | 1. 教师登录<br>2. 查看学生错题<br>3. 添加评论批注<br>4. 学生接收通知 | 批注成功添加并通知 |

### 7.2 性能测试用例
| 测试用例 ID | 测试场景 | 并发数 | 预期响应时间 |
|------------|----------|--------|--------------|
| TC-PERF-001 | Memo 列表查询 | 10并发 | < 500ms |
| TC-PERF-002 | 附件上传 | 5并发 | < 2s (1MB文件) |
| TC-PERF-003 | 全文搜索 | 20并发 | < 1s |

## 8. 安全测试用例

| 测试用例 ID | 测试场景 | 验证点 | 预期结果 |
|------------|----------|--------|----------|
| TC-SEC-001 | 未授权访问私有 Memo | 无 token 访问私有 Memo | 返回 401 Unauthorized |
| TC-SEC-002 | 跨用户数据访问 | 用户A尝试访问用户B的私有 Memo | 返回 403 Forbidden |
| TC-SEC-003 | SQL 注入防护 | 在筛选器中输入 SQL 语句 | 返回 400 Bad Request |
| TC-SEC-004 | 文件上传安全 | 上传可执行文件 | 返回 400 Bad Request |
| TC-SEC-005 | Token 过期验证 | 使用过期 token 访问 | 返回 401 Unauthorized |

## 9. 自动化测试脚本示例

```bash
#!/bin/bash
# 基础环境测试脚本

BASE_URL="http://localhost:5230"
JWT_TOKEN=""

# 1. 登录获取 token
login() {
    response=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin"}')
    JWT_TOKEN=$(echo $response | jq -r '.token')
    echo "Token: $JWT_TOKEN"
}

# 2. 测试健康检查
health_check() {
    curl -s "$BASE_URL/api/v1/health" | jq .
}

# 3. 创建测试 Memo
create_memo() {
    curl -s -X POST "$BASE_URL/api/v1/memos" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "content": "测试错题：二次函数求根公式",
            "visibility": "PRIVATE",
            "tags": ["数学", "函数"]
        }' | jq .
}

# 4. 运行测试套件
run_tests() {
    login
    health_check
    create_memo
}

run_tests
```

## 10. 测试数据准备

### 10.1 初始测试用户
```json
{
  "username": "test_teacher",
  "password": "Test@123",
  "email": "teacher@test.com",
  "display_name": "测试教师",
  "role": "ADMIN"
}
```

### 10.2 测试错题数据
```json
{
  "content": "题目：求解方程 x² - 5x + 6 = 0\n\n解析：使用求根公式...",
  "visibility": "PROTECTED",
  "tags": ["数学", "代数", "方程"],
  "properties": {
    "subject": "数学",
    "chapter": "二次函数",
    "difficulty": 3,
    "mastery": "未掌握"
  }
}
```

## 11. 测试报告模板

| 测试项目 | 测试用例总数 | 通过数 | 失败数 | 通过率 | 备注 |
|----------|--------------|--------|--------|--------|------|
| 用户管理 | 8 | 8 | 0 | 100% | |
| Memo 管理 | 15 | 14 | 1 | 93.3% | 附件上传偶发失败 |
| 附件管理 | 4 | 4 | 0 | 100% | |
| 安全测试 | 5 | 5 | 0 | 100% | |
| 性能测试 | 3 | 2 | 1 | 66.7% | 并发查询需优化 |
| **总计** | **35** | **33** | **2** | **94.3%** | |

---
*测试用例设计基于 memos v0.26.1 API 规范*
*适用于错题本平台定制开发*