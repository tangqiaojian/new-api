/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import type {
  ChannelSubscriptionPool,
  SubscriptionResetScope,
} from '../../types'

interface ChannelPoolResetDialogProps {
  pool: ChannelSubscriptionPool | null
  scope: SubscriptionResetScope
  isLoading: boolean
  onScopeChange: (scope: SubscriptionResetScope) => void
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}

export function ChannelPoolResetDialog(props: ChannelPoolResetDialogProps) {
  const { t } = useTranslation()
  return (
    <ConfirmDialog
      open={props.pool !== null}
      onOpenChange={props.onOpenChange}
      title={t('Reset channel pool usage')}
      desc={t('Choose which usage counters to reset for {{name}}.', {
        name: props.pool?.name ?? '',
      })}
      confirmText={t('Reset usage')}
      isLoading={props.isLoading}
      handleConfirm={props.onConfirm}
    >
      <Select
        value={props.scope}
        onValueChange={(scope) => {
          if (scope) props.onScopeChange(scope)
        }}
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
    </ConfirmDialog>
  )
}

interface ChannelPoolCancelDialogProps {
  pool: ChannelSubscriptionPool | null
  isLoading: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}

export function ChannelPoolCancelDialog(props: ChannelPoolCancelDialogProps) {
  const { t } = useTranslation()
  return (
    <ConfirmDialog
      open={props.pool !== null}
      onOpenChange={props.onOpenChange}
      title={t('Cancel channel pool')}
      desc={t(
        'Cancel {{name}}? Its channels will be released and this action cannot be undone.',
        { name: props.pool?.name ?? '' }
      )}
      confirmText={t('Cancel pool')}
      destructive
      isLoading={props.isLoading}
      handleConfirm={props.onConfirm}
    />
  )
}
