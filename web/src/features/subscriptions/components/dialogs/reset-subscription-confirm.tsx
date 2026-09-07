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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { adminResetSubscription } from '../../api'
import type {
  AdminUserSubscriptionItem,
  SubscriptionResetScope,
} from '../../types'

interface ResetSubscriptionConfirmProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  subscription: AdminUserSubscriptionItem | null
  onSuccess: () => void
}

export function ResetSubscriptionConfirm({
  open,
  onOpenChange,
  subscription,
  onSuccess,
}: ResetSubscriptionConfirmProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [resetScope, setResetScope] = useState<SubscriptionResetScope>('both')
  const [advanceResetTime, setAdvanceResetTime] = useState(false)

  useEffect(() => {
    if (open) {
      setResetScope('both')
      setAdvanceResetTime(false)
    }
  }, [open])

  const handleConfirm = async () => {
    if (!subscription) return
    setLoading(true)
    try {
      const result = await adminResetSubscription(subscription.id, {
        reset_scope: resetScope,
        advance_reset_time: advanceResetTime,
      })
      if (result.success) {
        toast.success(t('Subscription usage has been reset'))
        onOpenChange(false)
        onSuccess()
      } else {
        toast.error(result.message || t('Operation failed'))
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Reset Subscription?')}
      desc={t(
        'Reset this subscription usage. Last reset time will update immediately. Optionally roll the billing period forward.'
      )}
      confirmText={t('Reset Usage')}
      destructive
      handleConfirm={handleConfirm}
      isLoading={loading}
      disabled={!subscription}
    >
      <div className='space-y-3'>
        <Select
          value={resetScope}
          onValueChange={(value) =>
            setResetScope(value as SubscriptionResetScope)
          }
        >
          <SelectTrigger aria-label={t('Reset scope')}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='quota'>{t('Quota only')}</SelectItem>
            <SelectItem value='tokens'>{t('Tokens only')}</SelectItem>
            <SelectItem value='both'>{t('Quota and tokens')}</SelectItem>
          </SelectContent>
        </Select>
        <div className='flex items-start gap-2'>
          <Checkbox
            id='advance-reset-time-single'
            checked={advanceResetTime}
            onCheckedChange={(checked) =>
              setAdvanceResetTime(checked === true)
            }
            className='mt-0.5'
          />
          <Label
            htmlFor='advance-reset-time-single'
            className='text-sm leading-5 font-normal'
          >
            {t('Also roll to next billing period')}
          </Label>
        </div>
      </div>
    </ConfirmDialog>
  )
}
