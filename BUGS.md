# BUGS

| ID | 状态 | 描述 | 证据 |
|----|------|------|------|
| B1 | fixed | /api/data 缺 token 拆分 | usedata.go aggregate + LogQuotaData 入账；usedata_token_split_test.go |
| B2 | fixed | SumUsedQuota 二次 Scan 覆盖 quota | log.go 分结构体 Scan；log_stat_test.go |
| B3 | in_progress | 用户维度 KPI；详情下钻仍缺 | UsageKpiGrid 已挂 users 区；待单用户详情 |
| B4 | fixed | 手动重置不写 LastResetTime；weekly=0 | subscription.go + UI + weekly sync |
| B5 | open | 看板交互（空态/时区/金额/Tab/筛选） | 待逐条记录 |

## B5 明细

（迭代中追加）
