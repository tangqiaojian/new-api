# PLAN — Sub2API 风格运营仪表盘

## 目标

把 New-API 控制台改成 Sub2API 信息架构：8 张小图标卡 + 分组/渠道卡 + 图表区。
Overview / Dashboard / 用户详情复用同一套 Stats。**禁止**超大数字四宫格。

## 完成证据（2026-09-07）

- 代码：Overview / Models(LogStatCards) / Users 均挂 OpsStatsGrid +（选用户后）OpsGroupCards + 甜甜圈/Token 趋势
- 后端：standard_quota = quota / group_ratio（忽略 user_group_ratio<=0 哨兵）；预览库近 1 日 actual 1.575M / standard 31.5M
- 前端测试：8 卡标题、暗色 *-900/30+*-400、分组卡 Never reset
- 预览：http://192.168.6.88:3000 develop-dashboard healthy
- P0：LastResetTime / token 拆分 / 用户排行仍在

## 完成标准

- [x] Overview 8 张小图标卡
- [x] Token 卡一行输入/输出/缓存
- [x] 今日与累计
- [x] 分组卡 + 订阅进度/重置
- [x] 点排行进同一套仪表盘（含分组卡+图表）
- [x] 信息架构对齐 Sub2API（暗色图标色 + 实际/标准价）
- [x] 今日费用实际 / 标准价（group_ratio 反推）
- [x] P0 订阅重置 / token 拆分 / 用户排行
