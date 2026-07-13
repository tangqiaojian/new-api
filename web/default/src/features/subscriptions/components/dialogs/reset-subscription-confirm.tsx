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
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'

import { adminResetSubscription } from '../../api'
import type { AdminUserSubscriptionItem } from '../../types'

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

  const handleConfirm = async () => {
    if (!subscription) return
    setLoading(true)
    try {
      const result = await adminResetSubscription(subscription.id)
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
        'Are you sure you want to reset this subscription? The next reset time will be recalculated according to the plan period.'
      )}
      confirmText={t('Reset Usage')}
      destructive
      handleConfirm={handleConfirm}
      isLoading={loading}
      disabled={!subscription}
    />
  )
}
