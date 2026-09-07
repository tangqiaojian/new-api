# PLAN — Sub2API 风格运营仪表盘

## 目标

把 New-API 控制台改成 Sub2API 信息架构：8 张小图标卡 + 分组/渠道卡 + 图表区。
Overview / Dashboard / 用户详情复用同一套 Stats。**禁止**超大数字四宫格。

## 已完成

- iter1：OpsStatsGrid 8 小卡；UsageKpiGrid 降级
- iter2：`GET /api/data/groups`；OpsGroupCards + 订阅进度/重置
- iter3：用户排行点击 → 同一套 OpsStats 详情
- iter4：Models 区 Token 趋势（输入/输出/缓存三系列）
- 预览：http://192.168.6.88:3000 `develop-dashboard`

## 缺口

- 模型甜甜圈「可切用户消耗排行」与 Sub2API 完全同构（现有 Models 比例图 + Users 排行分 Tab）
- 今日费用实际价 vs 标准价（暂同值）
- 截图对照 Sub2API 密度/对齐做最后微调

## 完成标准

- [x] Overview 8 张小图标卡（非巨无霸四宫格）
- [x] Token 卡一行输入/输出/缓存
- [x] 今日与累计
- [x] 分组卡 + 订阅进度/重置（用户端）
- [x] 点排行进同一套仪表盘
- [ ] 截图对照 Sub2API（待你目视确认）
- [x] P0 订阅重置 / token 拆分 / 用户排行
