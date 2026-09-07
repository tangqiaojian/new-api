# PLAN — 仅前端统计 UI 靠齐 Sub2API

## 目标

回滚导致各界面报错的 Goal2 OpsStats/后端看板改动；**只做前端**统计卡视觉向 Sub2API 靠齐（左图标方块 + 紧凑字号 + 暗色 `*-900/30`/`*-400`），不新增后端字段。

## 做法

1. 代码回滚到 `2d6617f`（Goal1 完成态）相关看板文件，删除 Ops* 组件与 groups/standard_quota 等
2. 重写 `UsageKpiGrid` 为紧凑 Sub2API 风格（仍用原有 token 拆分数据）
3. typecheck + 热更预览站
