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
import { api } from '@/lib/api'
import type {
  FlowQuotaDataItem,
  QuotaDataItem,
  UptimeGroupResult,
  DailyTokenDataItem,
  DailyModelTokenDataItem,
} from './types'

// ============================================================================
// Dashboard APIs
// ============================================================================

// ----------------------------------------------------------------------------
// Quota & Usage Data
// ----------------------------------------------------------------------------

// Get user quota data within a time range
// Admin users get all users' data by default (matching classic frontend behavior)
export async function getUserQuotaDates(
  params: {
    start_timestamp: number
    end_timestamp: number
    default_time?: string
    username?: string
  },
  isAdmin = false
) {
  const endpoint = isAdmin ? '/api/data' : '/api/data/self'
  const res = await api.get<{ success: boolean; data: QuotaDataItem[] }>(
    endpoint,
    { params }
  )
  return res.data
}

// ----------------------------------------------------------------------------
// System Monitoring
// ----------------------------------------------------------------------------

export async function getUserQuotaDataByUsers(params: {
  start_timestamp: number
  end_timestamp: number
}) {
  const res = await api.get<{ success: boolean; data: QuotaDataItem[] }>(
    '/api/data/users',
    { params }
  )
  return res.data
}

export async function getFlowQuotaDates(
  params: {
    start_timestamp: number
    end_timestamp: number
    default_time?: string
    username?: string
  },
  isAdmin = false
) {
  const endpoint = isAdmin ? '/api/data/flow' : '/api/data/flow/self'
  const res = await api.get<{
    success: boolean
    data?: FlowQuotaDataItem[]
    message?: string
  }>(endpoint, { params })
  return res.data
}

// Get uptime monitoring status for all services
export async function getUptimeStatus() {
  const res = await api.get<{ success: boolean; data: UptimeGroupResult[] }>(
    '/api/uptime/status'
  )
  return res.data
}

// ----------------------------------------------------------------------------
// Daily Token Usage Statistics
// ----------------------------------------------------------------------------

// Get daily token usage data for all users (admin only)
export async function getDailyTokenData(params: {
  start_timestamp: number
  end_timestamp: number
  username?: string
  include_cache?: boolean
}) {
  const res = await api.get<{ success: boolean; data: DailyTokenDataItem[] }>(
    '/api/data/daily-tokens',
    { params: { ...params, include_cache: params.include_cache !== false } }
  )
  return res.data
}

// Get daily token usage data for current user
export async function getSelfDailyTokenData(params: {
  start_timestamp: number
  end_timestamp: number
  include_cache?: boolean
}) {
  const res = await api.get<{ success: boolean; data: DailyTokenDataItem[] }>(
    '/api/data/daily-tokens/self',
    { params: { ...params, include_cache: params.include_cache !== false } }
  )
  return res.data
}

// ----------------------------------------------------------------------------
// Daily Model Token Usage Statistics
// ----------------------------------------------------------------------------

// Get daily model token usage data for all users (admin only)
export async function getDailyModelTokenData(params: {
  start_timestamp: number
  end_timestamp: number
  include_cache?: boolean
}) {
  const res = await api.get<{
    success: boolean
    data: DailyModelTokenDataItem[]
  }>('/api/data/daily-model-tokens', {
    params: { ...params, include_cache: params.include_cache !== false },
  })
  return res.data
}

// Get daily model token usage data for current user
export async function getSelfDailyModelTokenData(params: {
  start_timestamp: number
  end_timestamp: number
  include_cache?: boolean
}) {
  const res = await api.get<{
    success: boolean
    data: DailyModelTokenDataItem[]
  }>('/api/data/daily-model-tokens/self', {
    params: { ...params, include_cache: params.include_cache !== false },
  })
  return res.data
}

// ----------------------------------------------------------------------------
// Channel Statistics
// ----------------------------------------------------------------------------

export interface ChannelStatsItem {
  channel_id: number
  channel_name: string
  request_count: number
  success_count: number
  error_count: number
  success_rate: number
  avg_use_time: number
  avg_first_byte: number
  total_tokens: number
  cached_tokens: number
  cache_hit_ratio: number
  used_quota: number
}

export interface ChannelStatsSummary {
  total_requests: number
  total_success: number
  total_errors: number
  overall_success_rate: number
  avg_use_time: number
  avg_first_byte: number
  total_tokens: number
  total_cached_tokens: number
  overall_cache_hit_ratio: number
  total_used_quota: number
  top_channels: ChannelStatsItem[]
  all_channels: ChannelStatsItem[]
}

export async function getChannelStats(params: {
  start_timestamp: number
  end_timestamp: number
  include_cache?: boolean
}) {
  const res = await api.get<{
    success: boolean
    data: ChannelStatsSummary
  }>('/api/data/channel-stats', {
    params: {
      ...params,
      include_cache: params.include_cache !== false,
    },
  })
  return res.data
}
