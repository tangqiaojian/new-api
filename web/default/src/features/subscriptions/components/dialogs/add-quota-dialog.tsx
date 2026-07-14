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

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  formatQuotaWithCurrency,
  getCurrencyDisplay,
  getCurrencyLabel,
} from '@/lib/currency'
import {
  formatChineseNumber,
  formatCompactNumber,
  parseQuotaFromDollars,
} from '@/lib/format'

import { adminAdjustSubscription } from '../../api'
import type { AdminUserSubscriptionItem } from '../../types'

interface AddQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  subscription: AdminUserSubscriptionItem | null
  onSuccess: () => void
}

export function AddQuotaDialog({
  open,
  onOpenChange,
  subscription,
  onSuccess,
}: AddQuotaDialogProps) {
  const { t } = useTranslation()
  const [amount, setAmount] = useState('')
  const [tokens, setTokens] = useState('')
  const [loading, setLoading] = useState(false)

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'

  // Reset inputs whenever the dialog is (re)opened.
  useEffect(() => {
    if (open) {
      setAmount('')
      setTokens('')
    }
  }, [open])

  const amountValue = parseFloat(amount) || 0
  const tokenValue = parseInt(tokens, 10) || 0

  // amount_delta is in quota units (dollars * quotaPerUnit).
  const amountDelta = parseQuotaFromDollars(amountValue)
  const tokenDelta = tokenValue

  const hasInput = amountDelta > 0 || tokenDelta > 0

  const handleConfirm = async () => {
    if (!subscription || !hasInput) return
    setLoading(true)
    try {
      const result = await adminAdjustSubscription(
        subscription.id,
        amountDelta,
        tokenDelta
      )
      if (result.success) {
        toast.success(t('Quota adjusted successfully'))
        onOpenChange(false)
        onSuccess()
      } else {
        toast.error(result.message || t('Failed to adjust quota'))
      }
    } catch (e: unknown) {
      toast.error(
        e instanceof Error ? e.message : t('Failed to adjust quota')
      )
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    onOpenChange(false)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Add Quota')}
      description={t(
        'Add quota to the subscription for {{username}} ({{plan}})',
        {
          username: subscription?.username ?? '-',
          plan: subscription?.plan_title ?? '-',
        }
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={handleCancel}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={loading || !hasInput}>
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <div className='space-y-2'>
          <Label>
            {t('Add Amount')} ({currencyLabel})
          </Label>
          <Input
            type='number'
            step={tokensOnly ? 1 : 0.01}
            min={0}
            placeholder={t('Enter amount in {{currency}}', {
              currency: currencyLabel,
            })}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <p className='text-muted-foreground text-xs'>
            {amountDelta > 0
              ? `${t('Adds')} ${formatQuotaWithCurrency(amountDelta)}`
              : t('0 means no change')}
          </p>
        </div>

        <div className='space-y-2'>
          <Label>{t('Add Tokens')}</Label>
          <Input
            type='number'
            step={1}
            min={0}
            placeholder={t('Enter token count')}
            value={tokens}
            onChange={(e) => setTokens(e.target.value)}
          />
          <p className='text-muted-foreground text-xs'>
            {tokenDelta > 0
              ? `${t('Adds')} ${formatCompactNumber(tokenDelta)} ${t('tokens')}（${formatChineseNumber(tokenDelta)}）`
              : t('0 means no change')}
          </p>
        </div>
      </div>
    </Dialog>
  )
}
