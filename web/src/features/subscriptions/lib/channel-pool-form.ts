/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { TFunction } from 'i18next'
import { z } from 'zod'

import type { ChannelPoolPayload, ChannelSubscriptionPool } from '../types'

export function getChannelPoolFormSchema(t: TFunction) {
  return z.object({
    name: z.string().trim().min(1, t('Please enter a pool name')).max(128),
    plan_id: z.coerce.number().int().positive(t('Please select a plan')),
    channel_ids: z
      .array(z.coerce.number().int().nonnegative())
      .min(1, t('Please select at least one channel')),
    quota_reset_anchor: z.date().optional(),
    quota_reset_timezone: z.string(),
    token_reset_anchor: z.date().optional(),
    token_reset_timezone: z.string(),
  })
}

export type ChannelPoolFormValues = z.infer<
  ReturnType<typeof getChannelPoolFormSchema>
>

export const CHANNEL_POOL_FORM_DEFAULTS: ChannelPoolFormValues = {
  name: '',
  plan_id: 0,
  channel_ids: [],
  quota_reset_anchor: undefined,
  quota_reset_timezone: '',
  token_reset_anchor: undefined,
  token_reset_timezone: '',
}

export function channelPoolToFormValues(
  pool: ChannelSubscriptionPool
): ChannelPoolFormValues {
  return {
    name: pool.name,
    plan_id: pool.plan_id,
    channel_ids: (pool.channels ?? []).map((member) => member.channel_id),
    quota_reset_anchor:
      pool.quota_reset_anchor && pool.quota_reset_anchor > 0
        ? new Date(pool.quota_reset_anchor * 1000)
        : undefined,
    quota_reset_timezone: pool.quota_reset_timezone ?? '',
    token_reset_anchor:
      pool.token_reset_anchor && pool.token_reset_anchor > 0
        ? new Date(pool.token_reset_anchor * 1000)
        : undefined,
    token_reset_timezone: pool.token_reset_timezone ?? '',
  }
}

export function channelPoolFormToPayload(
  values: ChannelPoolFormValues
): ChannelPoolPayload {
  return {
    name: values.name.trim(),
    plan_id: values.plan_id,
    channel_ids: values.channel_ids,
    quota_reset_anchor: values.quota_reset_anchor
      ? Math.floor(values.quota_reset_anchor.getTime() / 1000)
      : null,
    quota_reset_timezone: values.quota_reset_anchor
      ? values.quota_reset_timezone || null
      : null,
    token_reset_anchor: values.token_reset_anchor
      ? Math.floor(values.token_reset_anchor.getTime() / 1000)
      : null,
    token_reset_timezone: values.token_reset_anchor
      ? values.token_reset_timezone || null
      : null,
  }
}
