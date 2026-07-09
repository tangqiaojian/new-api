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
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { type Table } from '@tanstack/react-table'
import { Power, PowerOff, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { batchDeleteUsers, batchManageUsers } from '../api'
import { type User } from '../types'

interface DataTableBulkActionsProps {
  table: Table<User>
}

export function DataTableBulkActions({ table }: DataTableBulkActionsProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [loading, setLoading] = useState(false)

  const selectedRows = table.getFilteredSelectedRowModel().rows
  const selectedIds = selectedRows.reduce<number[]>((ids, row) => {
    const id = (row.original as User).id

    if (typeof id === 'number') {
      ids.push(id)
    }

    return ids
  }, [])

  const handleClearSelection = () => {
    table.resetRowSelection()
  }

  const handleEnableAll = async () => {
    if (selectedIds.length === 0) return
    setLoading(true)
    try {
      const result = await batchManageUsers(selectedIds, 'enable')
      if (result.success) {
        toast.success(
          t('Enabled {{count}} users', { count: selectedIds.length })
        )
        void queryClient.invalidateQueries({ queryKey: ['users'] })
        handleClearSelection()
      } else {
        toast.error(result.message || t('Failed to enable users'))
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('Failed to enable users'))
    } finally {
      setLoading(false)
    }
  }

  const handleDisableAll = async () => {
    if (selectedIds.length === 0) return
    setLoading(true)
    try {
      const result = await batchManageUsers(selectedIds, 'disable')
      if (result.success) {
        toast.success(
          t('Disabled {{count}} users', { count: selectedIds.length })
        )
        void queryClient.invalidateQueries({ queryKey: ['users'] })
        handleClearSelection()
      } else {
        toast.error(result.message || t('Failed to disable users'))
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t('Failed to disable users')
      )
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteAll = async () => {
    if (selectedIds.length === 0) return
    setLoading(true)
    try {
      const result = await batchDeleteUsers(selectedIds)
      if (result.success) {
        toast.success(
          t('Deleted {{count}} users', { count: selectedIds.length })
        )
        setShowDeleteConfirm(false)
        void queryClient.invalidateQueries({ queryKey: ['users'] })
        handleClearSelection()
      } else {
        toast.error(result.message || t('Failed to delete users'))
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t('Failed to delete users')
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <BulkActionsToolbar table={table} entityName='user'>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant='outline'
                size='icon'
                onClick={handleEnableAll}
                disabled={loading}
                className='size-8'
                aria-label={t('Enable selected users')}
                title={t('Enable selected users')}
              />
            }
          >
            <Power />
            <span className='sr-only'>{t('Enable selected users')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Enable selected users')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant='outline'
                size='icon'
                onClick={handleDisableAll}
                disabled={loading}
                className='size-8'
                aria-label={t('Disable selected users')}
                title={t('Disable selected users')}
              />
            }
          >
            <PowerOff />
            <span className='sr-only'>{t('Disable selected users')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Disable selected users')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant='destructive'
                size='icon'
                onClick={() => setShowDeleteConfirm(true)}
                disabled={loading}
                className='size-8'
                aria-label={t('Delete selected users')}
                title={t('Delete selected users')}
              />
            }
          >
            <Trash2 />
            <span className='sr-only'>{t('Delete selected users')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Delete selected users')}</p>
          </TooltipContent>
        </Tooltip>
      </BulkActionsToolbar>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title={t('Delete Users?')}
        description={t(
          'Are you sure you want to delete {{count}} selected users? This action cannot be undone.',
          { count: selectedIds.length }
        )}
        contentHeight='auto'
        footer={
          <>
            <Button
              variant='outline'
              onClick={() => setShowDeleteConfirm(false)}
            >
              {t('Cancel')}
            </Button>
            <Button
              variant='destructive'
              onClick={handleDeleteAll}
              disabled={loading}
            >
              {t('Delete')}
            </Button>
          </>
        }
      >
        {' '}
      </Dialog>
    </>
  )
}
