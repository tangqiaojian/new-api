# PLAN — 数据看板 + 用户用量 + 订阅手动重置

## 目标

把 New-API「数据看板 + 用户用量 + 订阅手动重置时间」做到可上线。数字先对，再对齐参考 UI。

## 当前迭代

**迭代 4**：B1 `/api/data` token 拆分（prompt/completion/cache_read/cache_write + success/error）

### 本轮目标

1. 扩展 QuotaData 与 logs 聚合 SELECT，返回拆分字段
2. LogQuotaData 写入路径带上 cache / prompt / completion
3. 单测：带 other.cache_tokens / cache_write_tokens 的 logs 聚合结果正确

### 已完成

- B4 手动重置写 LastResetTime；advance 可选；Never 周期不清除 last
- B4 UI「从未重置」+ 推进周期勾选；weekly_quota 同步
- B2 SumUsedQuota 二次 Scan 不再覆盖 quota

### 缺口

- B3 用户维度 KPI 页 + UsageKpiGrid
- B5 看板交互 bug
- 部署 192.168.6.88

### 风险

- cache_read 字段名为 other.cache_tokens；cache_write 为 other.cache_write_tokens
- 旧数据拆分为 0 可接受；新请求必须入账
- Get*QuotaDates 现查 logs 表，与 quota_data 缓存表并存

## 完成标准（待勾）

- [ ] B1 新请求入账 prompt/completion/cache_read/cache_write + success/error
- [x] B2 同范围顶部用量与 logs SUM 一致；rpm/tpm 不再被覆盖（后端单测绿）
- [ ] B3 /api/data/users 有数据；用户用量页 + 4 卡
- [x] B4 手动重置写 last_reset_time；UI 展示；weekly 对齐；单测绿
- [ ] B5 看板交互 bug 记入 BUGS.md 并修
- [ ] UsageKpiGrid 管理员总览 + 用户详情共用
- [ ] i18n zh-CN + en；暗色不崩
- [ ] 连续 2 迭代无新回归
- [ ] 部署到 192.168.6.88
