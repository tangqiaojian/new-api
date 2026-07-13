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
import { type ColumnDef } from '@tanstack/react-table'
import { X as Cross2Icon } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DataTablePage,
  useDataTable,
} from '@/components/data-table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useDebounce } from '@/hooks'

import { adminListAllSubscriptions } from '../api'
import type { AdminUserSubscriptionItem } from '../types'
import { useUserSubscriptionsColumns } from './user-subscriptions-columns'

const DEFAULT_PAGE_SIZE = 10

const EMPTY_LIST = {
  items: [] as AdminUserSubscriptionItem[],
  total: 0,
  page: 1,
  page_size: DEFAULT_PAGE_SIZE,
}

interface UserSubscriptionsTableProps {
  onAddQuota: (row: AdminUserSubscriptionItem) => void
  onReset: (row: AdminUserSubscriptionItem) => void
  refreshTrigger: number
}

export function UserSubscriptionsTable({
  onAddQuota,
  onReset,
  refreshTrigger,
}: UserSubscriptionsTableProps) {
  const { t } = useTranslation()
  const [pageIndex, setPageIndex] = useState(0)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [searchInput, setSearchInput] = useState('')
  const [isComposing, setIsComposing] = useState(false)
  const debouncedSearch = useDebounce(searchInput, 400)

  // The effective username filter. While composing (IME), keep the previously
  // committed value so the list doesn't flicker mid-composition.
  const usernameFilter = isComposing ? undefined : debouncedSearch || undefined

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'admin-user-subscriptions',
      pageIndex,
      pageSize,
      usernameFilter,
      refreshTrigger,
    ],
    queryFn: async () => {
      const result = await adminListAllSubscriptions({
        p: pageIndex + 1,
        size: pageSize,
        username: usernameFilter,
      })
      if (!result?.success) {
        return EMPTY_LIST
      }
      return result.data || EMPTY_LIST
    },
    placeholderData: (prev) => prev,
  })

  const items = data?.items || []
  const total = data?.total || 0
  const pageCount = Math.max(1, Math.ceil(total / pageSize))

  const columns = useUserSubscriptionsColumns({ onAddQuota, onReset })

  // Keep the current page in range when the total shrinks (e.g. after a search).
  const ensurePageInRange = useCallback(
    (nextPageCount: number) => {
      if (pageIndex >= nextPageCount && nextPageCount > 0) {
        setPageIndex(nextPageCount - 1)
      }
    },
    [pageIndex]
  )

  const { table } = useDataTable({
    data: items,
    columns: columns as ColumnDef<AdminUserSubscriptionItem, unknown>[],
    manualPagination: true,
    manualFiltering: true,
    withFilteredRowModel: false,
    withFacetedRowModel: false,
    pagination: {
      pageIndex,
      pageSize,
    },
    onPaginationChange: (updater) => {
      const next =
        typeof updater === 'function'
          ? updater({ pageIndex, pageSize })
          : updater
      setPageIndex(next.pageIndex)
      setPageSize(next.pageSize)
    },
    pageCount,
    totalCount: total,
    ensurePageInRange,
  })

  // Reset to first page when the committed search term changes.
  useEffect(() => {
    setPageIndex(0)
  }, [debouncedSearch])

  const hasFilters = !!searchInput

  const handleReset = () => {
    setSearchInput('')
    setPageIndex(0)
  }

  return (
    <DataTablePage
      table={table}
      columns={columns as ColumnDef<AdminUserSubscriptionItem, unknown>[]}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No subscriptions found')}
      emptyDescription={t(
        'User subscriptions will appear here once they subscribe to a plan.'
      )}
      skeletonKeyPrefix='user-subscriptions-skeleton'
      applyHeaderSize
      toolbar={
        <div className='flex flex-wrap items-center gap-2 sm:gap-3'>
          <Input
            placeholder={t('Search by username')}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onCompositionStart={() => setIsComposing(true)}
            onCompositionEnd={(e) => {
              setIsComposing(false)
              setSearchInput(e.currentTarget.value)
            }}
            className='w-full sm:w-[220px] lg:w-[260px]'
          />
          {hasFilters && (
            <Button
              variant='ghost'
              onClick={handleReset}
              className='text-muted-foreground hover:text-foreground gap-1 px-2'
            >
              {t('Reset')}
              <Cross2Icon className='size-3.5' />
            </Button>
          )}
        </div>
      }
    />
  )
}
