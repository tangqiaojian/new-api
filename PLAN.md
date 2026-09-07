# PLAN — Sub2API 风格运营仪表盘

## 目标

把 New-API 控制台改成 Sub2API 信息架构：8 张小图标卡 + 分组/渠道卡 + 图表区。
Overview / Dashboard / 用户详情复用同一套 Stats。**禁止**超大数字四宫格。

## 已完成（iter1–5）

- OpsStatsGrid 8 小卡（余额/Keys/今日请求/今日费用/今日Token/累计Token/RPM·TPM/平均耗时）
- OpsGroupCards + `/api/data/groups`；用户端订阅进度与重置时间
- 用户排行点击 → 同一 OpsStats 详情
- Token 趋势三系列；甜甜圈可切模型分布 / 用户排行并跳转 Users
- 预览 http://192.168.6.88:3000 `develop-dashboard`

## 仍待你目视确认

- 与 Sub2API 截图对照：密度、对齐、图标色（暗色 `*-900/30` + `*-400` 已对齐代码）

## 本轮新增

- 今日费用「实际 / 标准价」：后端从日志 `other.group_ratio`（优先 `user_group_ratio`）反推 `standard_quota`；UI 展示 `actual / standard`，有折扣时标准价划线

## 完成标准

- [x] Overview 8 张小图标卡
- [x] Token 卡一行输入/输出/缓存
- [x] 今日与累计
- [x] 分组卡 + 订阅进度/重置
- [x] 点排行进同一套仪表盘
- [ ] 截图对照 Sub2API（待确认）
- [x] 今日费用实际 / 标准价（group_ratio 反推）
- [x] P0 订阅重置 / token 拆分 / 用户排行
