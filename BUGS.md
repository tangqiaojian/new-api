# BUGS

| ID | 状态 | 描述 | 证据 |
|----|------|------|------|
| B1 | open | /api/data 缺 token 拆分；LogQuotaData 可能未写入 cache | 上游 #5842/#4442/#1663 |
| B2 | open | SumUsedQuota 二次 Scan 覆盖 quota；rpm/tpm 与区间 quota 混用 | issue #7106；log.go:705-711 |
| B3 | open | /api/data/users 返回 [] | 线上复现 |
| B4 | open | 手动重置不写 LastResetTime（仅 advance 时写） | subscription.go:1575-1582 |
| B5 | open | 看板交互（空态/时区/金额/Tab/筛选） | 待逐条记录 |

## B5 明细

（迭代中追加）
