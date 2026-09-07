# PLAN — Sub2API 风格运营仪表盘

## 目标

把 New-API 控制台改成 Sub2API 信息架构：8 张小图标卡（今日/累计）+ 分组/渠道卡 + 图表区。
**作废**上一轮超大数字四宫格。Overview / Dashboard / 用户详情复用同一套 Stats。

## 当前

**iter1 完成**：`OpsStatsGrid` + `OpsGroupCards` 已挂 Overview / Models / Users；`UsageKpiGrid` 降级包装。

## 下一轮

- 图表区：模型甜甜圈可切用户排行；Token 趋势叠输入/输出/缓存；Top 用户趋势；点用户进同一仪表盘
- 分组卡：按 `group`/`use_group` + 订阅日/周/月进度条与重置时间
- 今日费用标准价字段（若后端可区分）

## 完成标准（待勾）

- [x] Overview 打开先看到 8 张小图标卡（非巨无霸四宫格）
- [x] Token 卡一张内读完输入/输出/缓存
- [x] 今日与累计两套数
- [ ] 订阅进度条与重置时间（分组卡）完整
- [ ] 点排行进同一套仪表盘
- [ ] 截图对照 Sub2API 密度/对齐/图标色
- [x] P0 订阅重置 + token 拆分 + 用户排行接口（上轮保留）
