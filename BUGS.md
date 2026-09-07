# BUGS

| ID | 状态 | 描述 |
|----|------|------|
| S1 | fixed | 超大数字四宫格 UsageKpiGrid — 已降级为 OpsStatsGrid 包装，Overview/Dashboard/用户区改用 8 张小卡 |
| S2 | fixed | 图表区：甜甜圈切用户排行、Token 趋势 Input/Output/Cache、Overview 同步；Top 用户趋势仍可增强 |
| S3 | fixed | 分组卡：/api/data/groups 按 use_group；用户端挂订阅进度与重置时间 |
| S4 | fixed | 今日费用实际/标准价：`standard_quota` = quota / group_ratio（优先 user_group_ratio）；UI 折扣划线 |
| B4 | fixed | 订阅手动重置 LastResetTime（上轮保留） |
| B1 | fixed | /api/data token 拆分（上轮保留） |
| B2 | fixed | SumUsedQuota Scan（上轮保留）；本轮补 avg_use_time |
| B3 | fixed | /api/data/users 非空（上轮保留） |
| S5 | open | 截图目视与 Sub2API 1:1 密度/对齐仍待确认 |
