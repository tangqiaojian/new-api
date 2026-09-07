# PLAN — 数据看板 + 用户用量 + 订阅手动重置

## 目标

把 New-API「数据看板 + 用户用量 + 订阅手动重置时间」做到可上线。数字先对，再对齐参考 UI。

## 当前迭代

**迭代 9+**：剩余 B5（时区/筛选卡片）、用户详情趋势打磨、连续无回归验证

### 已完成

- B4 手动重置写 LastResetTime；advance 可选；Never 周期不清除 last；weekly 同步；UI「从未重置」
- B2 SumUsedQuota 二次 Scan 不再覆盖 quota
- B1 /api/data token 拆分 + LogQuotaData 入账；UsageKpiGrid 挂 models/users
- B3 用户选择器切换四卡 KPI；空态
- B5-amount-zero / B5-empty / 时间快捷 Today·7·30
- 部署 192.168.6.88：version=develop-dashboard，docker healthy

### 缺口

- B5-timezone / B5-tab-label / B5-filter-cards 待线上对照
- 用户详情：模型拆分 + 按日趋势（单用户）
- 连续 2 迭代无新回归后才能勾完成

### 风险

- 部署未 push；二进制本地 gitignore
- 线上需登录才能验看板；勿改生产库/密码

## 完成标准（待勾）

- [x] B1 新请求入账 prompt/completion/cache_read/cache_write + success/error
- [x] B2 同范围顶部用量与 logs SUM 一致；rpm/tpm 不再被覆盖（后端单测绿）
- [x] B3 /api/data/users 聚合字段 + 用户区 KPI（详情选择器）
- [x] B4 手动重置写 last_reset_time；UI 展示；weekly 对齐；单测绿
- [ ] B5 看板交互 bug 全部关闭
- [x] UsageKpiGrid 管理员总览(models) + 用户区共用
- [x] i18n zh-CN + en；暗色用 card token
- [ ] 连续 2 迭代无新回归
- [x] 部署到 192.168.6.88（develop-dashboard healthy）
