# PLAN — Sub2API 风格运营仪表盘

## 目标

把 New-API 控制台改成 Sub2API 信息架构：8 张小图标卡 + 分组/渠道卡 + 图表区。
Overview / Dashboard / 用户详情复用同一套 Stats。**禁止**超大数字四宫格。

## 已完成

- iter1：OpsStatsGrid 8 小卡；UsageKpiGrid 降级
- iter2：`GET /api/data/groups` / `groups/self`；OpsGroupCards + 订阅进度/重置

## 下一轮

- 图表区：模型甜甜圈可切用户排行；Token 趋势叠输入/输出/缓存；Top 用户趋势；点用户进同一仪表盘
- 今日费用标准价（若可区分）
- 部署 192.168.6.88 对照截图

## 完成标准

- [x] Overview 8 张小图标卡
- [x] Token 卡一行输入/输出/缓存
- [x] 今日与累计
- [x] 分组卡 + 订阅进度/重置（用户端有订阅时）
- [ ] 点排行进同一套仪表盘（图表交互）
- [ ] 截图对照 Sub2API
- [x] P0 订阅重置 / token 拆分 / 用户排行
