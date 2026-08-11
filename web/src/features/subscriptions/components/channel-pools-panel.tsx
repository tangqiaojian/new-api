/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, RefreshCw, XCircle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getChannels } from '@/features/channels/api'
import { getCurrencyLabel } from '@/lib/currency'
import { formatNumber, formatQuota } from '@/lib/format'

import { cancelChannelPool, listChannelPools, resetChannelPool } from '../api'
import { formatTimestamp, isChannelPoolRoutable } from '../lib/format'
import type { ChannelSubscriptionPool, SubscriptionResetScope } from '../types'
import { ChannelPoolDrawer } from './channel-pool-drawer'
import {
  ChannelPoolCancelDialog,
  ChannelPoolResetDialog,
} from './dialogs/channel-pool-dialogs'

const PAGE_SIZE = 10

function usagePercent(used: number, total: number): number {
  if (total <= 0) return 0
  return Math.min(100, Math.max(0, (used / total) * 100))
}

function statusBadgeVariant(status: ChannelSubscriptionPool['status']) {
  if (status === 'active') return 'default'
  if (status === 'cancelled') return 'destructive'
  return 'secondary'
}

function statusLabel(
  status: ChannelSubscriptionPool['status'],
  t: ReturnType<typeof useTranslation>['t']
): string {
  if (status === 'active') return t('Active')
  if (status === 'expired') return t('Expired')
  return t('Cancelled')
}

export function ChannelPoolsPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editingPool, setEditingPool] =
    useState<ChannelSubscriptionPool | null>(null)
  const [resetPool, setResetPool] = useState<ChannelSubscriptionPool | null>(
    null
  )
  const [cancelPool, setCancelPool] = useState<ChannelSubscriptionPool | null>(
    null
  )
  const [resetScope, setResetScope] = useState<SubscriptionResetScope>('both')

  const poolsQuery = useQuery({
    queryKey: ['admin-channel-pools', page],
    queryFn: async () =>
      (await listChannelPools({ p: page, size: PAGE_SIZE })).data,
    placeholderData: (previous) => previous,
  })
  const channelsQuery = useQuery({
    queryKey: ['channels', 'channel-pool-member-names'],
    queryFn: async () =>
      (await getChannels({ p: 1, page_size: 100 })).data?.items ?? [],
  })

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: ['admin-channel-pools'] })
  const resetMutation = useMutation({
    mutationFn: () => {
      if (!resetPool) throw new Error('No channel pool selected')
      return resetChannelPool(resetPool.id, resetScope)
    },
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Reset succeeded'))
      setResetPool(null)
      void refresh()
    },
    onError: () => toast.error(t('Request failed')),
  })
  const cancelMutation = useMutation({
    mutationFn: () => {
      if (!cancelPool) throw new Error('No channel pool selected')
      return cancelChannelPool(cancelPool.id)
    },
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Channel pool cancelled'))
      setCancelPool(null)
      void refresh()
    },
    onError: () => toast.error(t('Request failed')),
  })

  const pools = poolsQuery.data?.items ?? []
  const channelNames = new Map(
    (channelsQuery.data ?? []).map((channel) => [channel.id, channel.name])
  )
  const currencyLabel = getCurrencyLabel()
  const totalPages = Math.max(
    1,
    Math.ceil((poolsQuery.data?.total ?? 0) / PAGE_SIZE)
  )

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <div className='flex justify-end'>
        <Button
          size='sm'
          onClick={() => {
            setEditingPool(null)
            setDrawerOpen(true)
          }}
        >
          <Plus className='h-4 w-4' />
          {t('Create channel pool')}
        </Button>
      </div>
      <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Pool')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Routable')}</TableHead>
              <TableHead>{t('Members')}</TableHead>
              <TableHead>
                {t('Quota ({{currency}})', { currency: currencyLabel })}
              </TableHead>
              <TableHead>{t('Token usage')}</TableHead>
              <TableHead>{t('Next reset')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pools.map((pool) => (
              <TableRow key={pool.id}>
                <TableCell>
                  <div className='font-medium'>{pool.name}</div>
                  <div className='text-muted-foreground text-xs'>
                    {pool.plan_title} · #{pool.id}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant={statusBadgeVariant(pool.status)}>
                    {statusLabel(pool.status, t)}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge
                    variant={
                      isChannelPoolRoutable(pool) ? 'default' : 'secondary'
                    }
                  >
                    {isChannelPoolRoutable(pool)
                      ? t('Routable')
                      : t('Not routable')}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div>
                    {t('{{count}} channels', {
                      count: (pool.channels ?? []).length,
                    })}
                  </div>
                  <div
                    className='text-muted-foreground max-w-48 truncate text-xs'
                    title={(pool.channels ?? [])
                      .map((member) => {
                        const name = channelNames.get(member.channel_id)
                        return name
                          ? `${name} (#${member.channel_id})`
                          : `#${member.channel_id}`
                      })
                      .join(', ')}
                  >
                    {(pool.channels ?? [])
                      .map((member) => {
                        const name = channelNames.get(member.channel_id)
                        return name
                          ? `${name} (#${member.channel_id})`
                          : `#${member.channel_id}`
                      })
                      .join(', ') || '-'}
                  </div>
                </TableCell>
                <TableCell className='min-w-40'>
                  <div className='mb-1 text-xs'>
                    {formatQuota(pool.amount_used)} /{' '}
                    {pool.amount_total > 0
                      ? formatQuota(pool.amount_total)
                      : t('Unlimited')}
                  </div>
                  <Progress
                    value={usagePercent(pool.amount_used, pool.amount_total)}
                  />
                </TableCell>
                <TableCell className='min-w-40'>
                  <div className='mb-1 text-xs'>
                    {formatNumber(pool.tokens_used)} /{' '}
                    {pool.tokens_total > 0
                      ? formatNumber(pool.tokens_total)
                      : t('Unlimited')}
                  </div>
                  <Progress
                    value={usagePercent(pool.tokens_used, pool.tokens_total)}
                  />
                </TableCell>
                <TableCell>
                  <div className='text-xs'>
                    {t('Quota')}: {formatTimestamp(pool.quota_next_reset_time)}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Tokens')}: {formatTimestamp(pool.token_next_reset_time)}
                  </div>
                </TableCell>
                <TableCell>
                  <div className='flex justify-end gap-1'>
                    <Button
                      size='icon-sm'
                      variant='ghost'
                      aria-label={t('Edit channel pool')}
                      disabled={pool.status !== 'active'}
                      onClick={() => {
                        setEditingPool(pool)
                        setDrawerOpen(true)
                      }}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      size='icon-sm'
                      variant='ghost'
                      aria-label={t('Reset channel pool usage')}
                      disabled={pool.status !== 'active'}
                      onClick={() => {
                        setResetScope('both')
                        setResetPool(pool)
                      }}
                    >
                      <RefreshCw />
                    </Button>
                    <Button
                      size='icon-sm'
                      variant='ghost'
                      aria-label={t('Cancel channel pool')}
                      disabled={pool.status !== 'active'}
                      onClick={() => setCancelPool(pool)}
                    >
                      <XCircle />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
            {!poolsQuery.isLoading && pools.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={8}
                  className='text-muted-foreground h-32 text-center'
                >
                  {t('No channel pools yet')}
                </TableCell>
              </TableRow>
            ) : null}
          </TableBody>
        </Table>
      </div>
      <div className='flex items-center justify-between text-sm'>
        <span className='text-muted-foreground'>
          {t('{{count}} channel pools', { count: poolsQuery.data?.total ?? 0 })}
        </span>
        <div className='flex items-center gap-2'>
          <Button
            variant='outline'
            size='sm'
            disabled={page <= 1}
            onClick={() => setPage((value) => value - 1)}
          >
            {t('Previous')}
          </Button>
          <span>
            {t('Page {{page}} of {{total}}', { page, total: totalPages })}
          </span>
          <Button
            variant='outline'
            size='sm'
            disabled={page >= totalPages}
            onClick={() => setPage((value) => value + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
      <ChannelPoolDrawer
        open={drawerOpen}
        pool={editingPool}
        onOpenChange={setDrawerOpen}
        onSuccess={() => void refresh()}
      />
      <ChannelPoolResetDialog
        pool={resetPool}
        scope={resetScope}
        isLoading={resetMutation.isPending}
        onScopeChange={setResetScope}
        onOpenChange={(open) => !open && setResetPool(null)}
        onConfirm={() => resetMutation.mutate()}
      />
      <ChannelPoolCancelDialog
        pool={cancelPool}
        isLoading={cancelMutation.isPending}
        onOpenChange={(open) => !open && setCancelPool(null)}
        onConfirm={() => cancelMutation.mutate()}
      />
    </div>
  )
}
