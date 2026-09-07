# PLAN — 数据看板 + 用户用量 + 订阅手动重置

## 目标

把 New-API「数据看板 + 用户用量 + 订阅手动重置时间」做到可上线。数字先对，再对齐参考 UI。

## 当前迭代

**迭代 1**：B4 手动重置写 LastResetTime + B2 SumUsedQuota 二次 Scan 覆盖

### 本轮目标

1. B4：`resetUserSubscriptionTx` 手动重置用量时始终写 `LastResetTime=now`；仅 `advanceResetTime` 时推进 `NextResetTime`
2. B4：补单测（不推进 / 推进）
3. B2：修 `SumUsedQuota` 二次 Scan 覆盖 quota；补单测

### 缺口

- weekly_quota_reset_at 与订阅 LastResetTime 对齐（迭代 2–3）
- 管理端 UI「从未重置」展示与刷新（迭代 2–3）
- B1 token 拆分（迭代 4–6）
- B3 用户维度 + UsageKpiGrid（迭代 7–9）

### 风险

- 自动周期重置路径也调用 `resetUserSubscriptionTx(..., true, ...)`，改默认行为时勿破坏自动推进
- SumUsedQuota 的 rpm/tpm 刻意只看最近 60 秒；quota 是区间汇总，合并 Scan 时字段必须分离

## 完成标准（待勾）

- [ ] B1 新请求入账 prompt/completion/cache_read/cache_write + success/error
- [ ] B2 同范围顶部用量与 logs SUM 一致；rpm/tpm 不再被覆盖
- [ ] B3 /api/data/users 有数据；用户用量页 + 4 卡
- [ ] B4 手动重置写 last_reset_time；UI 展示；weekly 对齐；单测绿
- [ ] B5 看板交互 bug 记入 BUGS.md 并修
- [ ] UsageKpiGrid 管理员总览 + 用户详情共用
- [ ] i18n zh-CN + en；暗色不崩
- [ ] 连续 2 迭代无新回归
- [ ] 部署到 192.168.6.88
