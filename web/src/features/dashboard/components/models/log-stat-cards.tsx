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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'

import {
  getQuotaDataByGroups,
  getUserQuotaDates,
} from '@/features/dashboard/api'
import { OpsDistributionPie } from '@/features/dashboard/components/ops-distribution-pie'
import { OpsGroupCards } from '@/features/dashboard/components/ops-group-cards'
import { OpsStatsGrid } from '@/features/dashboard/components/ops-stats-grid'
import { OpsTokenTrendChart } from '@/features/dashboard/components/ops-token-trend-chart'
import { useAutoRefresh } from '@/features/dashboard/hooks/use-auto-refresh'
import { useOpsDashboardStats } from '@/features/dashboard/hooks/use-ops-dashboard-stats'
import { buildQueryParams, getDefaultDays } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  QuotaDataItem,
} from '@/features/dashboard/types'
import { ROLE } from '@/lib/roles'
import { computeTimeRange } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

interface LogStatCardsProps {
  filters?: DashboardFilters
  onDataUpdate?: (data: QuotaDataItem[], loading: boolean) => void
  includeCache?: boolean
  onSelectUser?: (username: string) => void
}

export function LogStatCards(props: LogStatCardsProps) {
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = !!(user?.role && user.role >= ROLE.ADMIN)
  const { refetchInterval } = useAutoRefresh()
  const username = props.filters?.username?.trim() || undefined

  const timeRange = useMemo(
    () =>
      computeTimeRange(
        getDefaultDays(props.filters?.time_granularity),
        props.filters?.start_timestamp,
        props.filters?.end_timestamp
      ),
    [
      props.filters?.end_timestamp,
      props.filters?.start_timestamp,
      props.filters?.time_granularity,
    ]
  )
  const queryParams = useMemo(
    () => buildQueryParams(timeRange, props.filters),
    [props.filters, timeRange]
  )

  const { stats: opsStats, loading: opsLoading } = useOpsDashboardStats({
    username,
    includeCache: props.includeCache,
    refetchInterval: refetchInterval || false,
  })

  const quotaQuery = useQuery({
    queryKey: [
      'dashboard',
      'models',
      'log-stats',
      queryParams,
      isAdmin,
      props.includeCache,
    ],
    queryFn: () =>
      getUserQuotaDates(
        { ...queryParams, include_cache: props.includeCache },
        isAdmin
      ),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const groupQuery = useQuery({
    queryKey: [
      'dashboard',
      'models',
      'group-cards',
      timeRange,
      username,
      isAdmin,
      props.includeCache,
    ],
    queryFn: () =>
      getQuotaDataByGroups(
        {
          start_timestamp: timeRange.start_timestamp,
          end_timestamp: timeRange.end_timestamp,
          username,
          include_cache: props.includeCache,
        },
        isAdmin
      ),
    staleTime: 60_000,
    refetchInterval: refetchInterval || undefined,
  })

  const data = useMemo(
    () => quotaQuery.data?.data ?? [],
    [quotaQuery.data?.data]
  )
  const onDataUpdate = props.onDataUpdate

  useEffect(() => {
    onDataUpdate?.(data, quotaQuery.isLoading)
  }, [data, onDataUpdate, quotaQuery.isLoading])

  const groupCards = useMemo(() => {
    const rows = groupQuery.data?.data ?? []
    return rows
      .map((item) => {
        const name = (item.use_group || '').trim() || 'default'
        return {
          name,
          todayCost: Number(item.quota) || 0,
          requests: Number(item.count) || 0,
          tokens:
            (Number(item.prompt_tokens) || 0) +
            (Number(item.completion_tokens) || 0) +
            (Number(item.cache_read_tokens) || 0),
        }
      })
      .sort((a, b) => b.todayCost - a.todayCost)
      .slice(0, 8)
  }, [groupQuery.data?.data])

  return (
    <div className='space-y-3'>
      <OpsStatsGrid loading={opsLoading} stats={opsStats} />
      <OpsGroupCards loading={groupQuery.isLoading} items={groupCards} />
      <div className='grid gap-3 lg:grid-cols-2'>
        <OpsDistributionPie
          data={data}
          loading={quotaQuery.isLoading}
          onSelectUser={props.onSelectUser}
        />
        <OpsTokenTrendChart data={data} loading={quotaQuery.isLoading} />
      </div>
    </div>
  )
}
