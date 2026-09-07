# BUGS

| ID | 状态 | 描述 | 证据 |
|----|------|------|------|
| B1 | fixed | /api/data 缺 token 拆分 | usedata_token_split_test.go |
| B2 | fixed | SumUsedQuota 二次 Scan 覆盖 quota | log_stat_test.go |
| B3 | fixed | 用户维度 KPI + 详情下钻 | UsageKpiGrid + 用户选择 + 模型拆分表 |
| B4 | fixed | 手动重置不写 LastResetTime；weekly=0 | subscription_reset_test.go |
| B5 | in_progress | 看板交互 | 见明细 |

## B5 明细

| 子项 | 状态 | 说明 |
|------|------|------|
| B5-amount-zero | fixed | charts Usage toFixed(6) |
| B5-empty | fixed | 用户区空态 |
| B5-timezone | fixed | dayjs 默认 Asia/Shanghai |
| B5-tab-label | open | 待对照参考图 |
| B5-filter-cards | open | queryKey 含 filters；待线上确认 |
| B5-presets | fixed | Today / 7 Days / 30 Days |
