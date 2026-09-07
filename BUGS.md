# BUGS

| ID | 状态 | 描述 |
|----|------|------|
| S1 | fixed | 超大数字四宫格 UsageKpiGrid — 已降级为 OpsStatsGrid 包装，Overview/Dashboard/用户区改用 8 张小卡 |
| S2 | open | 图表区：甜甜圈切用户排行、Token 趋势叠层、Top 用户趋势未对齐 Sub2API |
| S3 | open | 分组卡目前按 model_name 聚合；尚未接 use_group + 订阅限额进度 |
| S4 | open | 今日费用「实际/标准价」暂同值（网关暂无独立标准价字段） |
| B4 | fixed | 订阅手动重置 LastResetTime（上轮保留） |
| B1 | fixed | /api/data token 拆分（上轮保留） |
| B2 | fixed | SumUsedQuota Scan（上轮保留）；本轮补 avg_use_time |
| B3 | fixed | /api/data/users 非空（上轮保留） |
