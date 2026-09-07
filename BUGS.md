# BUGS

| ID | 状态 | 描述 | 证据 |
|----|------|------|------|
| B1 | fixed | /api/data 缺 token 拆分 | usedata.go aggregate + LogQuotaData；usedata_token_split_test.go |
| B2 | fixed | SumUsedQuota 二次 Scan 覆盖 quota | log.go 分结构体 Scan；log_stat_test.go |
| B3 | in_progress | 用户维度 KPI + 详情下钻 | UsageKpiGrid + 用户选择器已挂；排行点击可再增强 |
| B4 | fixed | 手动重置不写 LastResetTime；weekly=0 | subscription.go + UI + weekly sync |
| B5 | in_progress | 看板交互（空态/时区/金额/Tab/筛选） | 见明细 |

## B5 明细

| 子项 | 状态 | 说明 |
|------|------|------|
| B5-amount-zero | fixed | charts.ts Usage toFixed(4) 把小额打成 0 → 改为 toFixed(6) |
| B5-empty | fixed | 用户区空态文案 |
| B5-timezone | open | 待查 |
| B5-tab-label | open | 待对照参考图 |
| B5-filter-cards | open | LogStatCards queryKey 已含 filters；待线上复现 |
