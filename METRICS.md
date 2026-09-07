# METRICS

| iteration | time | fixed | tests | fail | notes |
|-----------|------|-------|-------|------|-------|
| 1 | 2026-09-07T13:48+08 | B4,B2 | SumUsedQuota*2 + AdminReset* | 0 | last_reset always stamped; Scan no longer wipes quota |
| 2-3 | 2026-09-07T13:53+08 | B4 UI,weekly | format-reset-time + weekly sync | 0 | Never reset i18n; advance_reset_time checkbox |
| 4-5 | 2026-09-07T14:00+08 | B1,+UsageKpiGrid | token-split*3 + stats-kpi*3 | 0 | /api/data returns prompt/completion/cache/success; KPI grid on models+users |
