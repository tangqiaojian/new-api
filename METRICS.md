# METRICS

| iteration | time | fixed | tests | fail | notes |
|-----------|------|-------|-------|------|-------|
| 1 | 2026-09-07T13:48+08 | B4,B2 | SumUsedQuota*2 + AdminReset* | 0 | last_reset always stamped; Scan no longer wipes quota |
| 2-3 | 2026-09-07T13:53+08 | B4 UI,weekly | format-reset-time + weekly sync | 0 | Never reset i18n; advance_reset_time checkbox |
| 4-5 | 2026-09-07T14:00+08 | B1,+UsageKpiGrid | token-split*3 + stats-kpi*3 | 0 | /api/data returns prompt/completion/cache/success; KPI grid |
| 6-7 | 2026-09-07T14:04+08 | B3,B5-amount | vitest 5 | 0 | user selector; Usage toFixed(6) |
| 8 | 2026-09-07T14:10+08 | B5-presets | — | 0 | Today/7/30 + first deploy develop-dashboard |
| 9 | 2026-09-07T14:12+08 | B5-tz,model | — | 0 | Asia/Shanghai; model breakdown |
| 10 | 2026-09-07T14:13+08 | — | go+vitest | 0 | clean regression + redeploy |
| 11 | 2026-09-07T14:15+08 | B5-tab | go+vitest | 0 | Consumption Trend tab+chart title; redeploy |
| 12 | 2026-09-07T14:20+08 | B4 user UI | go+vitest | 0 | user list/detail weekly reset display |
