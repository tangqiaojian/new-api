/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { OpsStatsGrid } from '@/features/dashboard/components/ops-stats-grid'
import {
  emptyOpsBucket,
  type OpsDashboardStats,
} from '@/features/dashboard/lib/stats'

export type UsageKpiStats = {
  totalCount: number
  promptTokens: number
  completionTokens: number
  cacheReadTokens: number
  successCount: number
}

interface UsageKpiGridProps {
  stats: UsageKpiStats | null
  loading?: boolean
  className?: string
}

/**
 * @deprecated Giant four-card KPI layout is retired. Maps legacy props into
 * Sub2API-style OpsStatsGrid so residual imports keep compiling.
 */
export function UsageKpiGrid(props: UsageKpiGridProps) {
  const today = props.stats
    ? {
        requests: props.stats.totalCount,
        quota: 0,
        promptTokens: props.stats.promptTokens,
        completionTokens: props.stats.completionTokens,
        cacheReadTokens: props.stats.cacheReadTokens,
        cacheWriteTokens: 0,
        tokenUsed:
          props.stats.promptTokens +
          props.stats.completionTokens +
          props.stats.cacheReadTokens,
      }
    : emptyOpsBucket()

  const mapped: OpsDashboardStats = {
    balanceQuota: 0,
    usedQuota: 0,
    apiKeysTotal: 0,
    apiKeysActive: 0,
    today,
    lifetime: emptyOpsBucket(),
    rpm: 0,
    tpm: 0,
    avgUseTimeSec: 0,
  }

  return (
    <OpsStatsGrid
      className={props.className}
      loading={props.loading}
      stats={mapped}
    />
  )
}
