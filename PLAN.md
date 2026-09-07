# PLAN — 数据看板 + 用户用量 + 订阅手动重置

## 目标

把 New-API「数据看板 + 用户用量 + 订阅手动重置时间」做到可上线。数字先对，再对齐参考 UI。

## 当前迭代

**迭代 11–12**：B5 收口 + 用户列表周额度重置展示 + 连续无回归验证 + 热更 192.168.6.88

### 已完成

- B4 手动重置写 LastResetTime；advance 可选；Never 周期不清除 last；weekly 同步；UI「从未重置」
- B2 SumUsedQuota 二次 Scan 不再覆盖 quota
- B1 /api/data token 拆分 + LogQuotaData 入账；UsageKpiGrid 挂 models/users
- B3 用户选择器切换四卡 KPI；模型拆分；空态；排行/趋势
- B5：小额 ¥0、空态、Asia/Shanghai、Today/7/30、Tab/图标题「消耗趋势」、筛选驱动 queryKey
- 用户列表/详情展示周额度下次重置（0 → 从未重置）
- 部署 192.168.6.88：version=develop-dashboard，docker healthy

### 缺口

- （可选）对账脚本 dashboard SUM vs logs SUM；大用户量聚合性能
- 连续 2 迭代无新回归：iter10 绿；iter12 验证中

### 风险

- 部署未 push；二进制本地 gitignore
- 线上需登录才能验看板；勿改生产库/密码

## 完成标准（待勾）

- [x] B1 新请求入账 prompt/completion/cache_read/cache_write + success/error
- [x] B2 同范围顶部用量与 logs SUM 一致；rpm/tpm 不再被覆盖（后端单测绿）
- [x] B3 /api/data/users 聚合字段 + 用户区 KPI（详情选择器）
- [x] B4 手动重置写 last_reset_time；UI 展示；weekly 对齐；单测绿
- [x] B5 看板交互 bug 全部关闭
- [x] UsageKpiGrid 管理员总览(models) + 用户区共用
- [x] i18n zh-CN + en；暗色用 card token
- [ ] 连续 2 迭代无新回归
- [x] 部署到 192.168.6.88（develop-dashboard healthy）
