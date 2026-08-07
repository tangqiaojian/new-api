/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { getChannels } from '@/features/channels/api'

import {
  createChannelPool,
  getAdminPlans,
  listChannelPoolOccupancies,
  updateChannelPool,
} from '../api'
import {
  CHANNEL_POOL_FORM_DEFAULTS,
  channelPoolFormToPayload,
  channelPoolToFormValues,
  getChannelPoolFormSchema,
  type ChannelPoolFormValues,
} from '../lib/channel-pool-form'
import type { ChannelSubscriptionPool } from '../types'

interface ChannelPoolDrawerProps {
  open: boolean
  pool: ChannelSubscriptionPool | null
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function ChannelPoolDrawer(props: ChannelPoolDrawerProps) {
  const { t } = useTranslation()
  const schema = getChannelPoolFormSchema(t)
  const form = useForm<ChannelPoolFormValues>({
    resolver: zodResolver(schema) as Resolver<ChannelPoolFormValues>,
    defaultValues: CHANNEL_POOL_FORM_DEFAULTS,
  })
  const plansQuery = useQuery({
    queryKey: ['admin-subscription-plans', 'channel-pool-options'],
    queryFn: async () => (await getAdminPlans()).data ?? [],
    enabled: props.open,
  })
  const channelsQuery = useQuery({
    queryKey: ['channels', 'channel-pool-options'],
    queryFn: async () =>
      (await getChannels({ p: 1, page_size: 100 })).data?.items ?? [],
    enabled: props.open,
  })
  const occupancyQuery = useQuery({
    queryKey: ['admin-channel-pool-occupancies'],
    queryFn: async () => (await listChannelPoolOccupancies()).data ?? [],
    enabled: props.open,
  })

  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.pool
        ? channelPoolToFormValues(props.pool)
        : CHANNEL_POOL_FORM_DEFAULTS
    )
  }, [form, props.open, props.pool])

  const planOptions = (plansQuery.data ?? []).filter(
    (record) =>
      record.plan.enabled &&
      (record.plan.plan_type === 'channel' || record.plan.plan_type === 'both')
  )
  const occupancyByChannel = new Map(
    (occupancyQuery.data ?? [])
      .filter((item) => !props.pool || item.pool_id !== props.pool.id)
      .map((item) => [item.channel_id, item.pool_name])
  )
  const channelOptions = (channelsQuery.data ?? []).map((channel) => {
    const occupiedBy = occupancyByChannel.get(channel.id)
    return {
      value: String(channel.id),
      label: `${channel.name} (#${channel.id})`,
      disabled: occupiedBy != null,
      disabledReason: occupiedBy
        ? t('Already in pool {{name}}', { name: occupiedBy })
        : undefined,
    }
  })

  const onSubmit = async (values: ChannelPoolFormValues) => {
    const payload = channelPoolFormToPayload(values)
    try {
      const response = props.pool
        ? await updateChannelPool(props.pool.id, payload)
        : await createChannelPool({ ...payload, plan_id: values.plan_id })
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(props.pool ? t('Update succeeded') : t('Create succeeded'))
      props.onOpenChange(false)
      props.onSuccess()
    } catch {
      toast.error(t('Request failed'))
    }
  }

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[600px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {props.pool ? t('Edit channel pool') : t('Create channel pool')}
          </SheetTitle>
          <SheetDescription>
            {t(
              'Assign channels to a channel subscription plan and track shared usage.'
            )}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='channel-pool-form'
            className={sideDrawerFormClassName()}
            onSubmit={form.handleSubmit(onSubmit)}
          >
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Pool name')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='plan_id'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Subscription plan')}</FormLabel>
                  <Select
                    value={field.value > 0 ? String(field.value) : ''}
                    onValueChange={(value) => field.onChange(Number(value))}
                    disabled={props.pool !== null}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Select a plan')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {planOptions.map((record) => (
                        <SelectItem
                          key={record.plan.id}
                          value={String(record.plan.id)}
                        >
                          {record.plan.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'Only enabled channel or user-and-channel plans are available.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='channel_ids'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Member channels')}</FormLabel>
                  <FormControl>
                    <MultiSelect
                      options={channelOptions}
                      selected={field.value.map(String)}
                      onChange={(values) => field.onChange(values.map(Number))}
                      placeholder={t('Search and select channels')}
                      emptyText={t('No matching channels')}
                      maxVisibleChips={5}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='quota_reset_anchor'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Quota reset anchor')}</FormLabel>
                    <FormControl>
                      <DateTimePicker
                        value={field.value}
                        onChange={(date) => {
                          field.onChange(date)
                          form.setValue(
                            'quota_reset_timezone',
                            date
                              ? Intl.DateTimeFormat().resolvedOptions().timeZone
                              : ''
                          )
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Leave empty to inherit the plan schedule.')}
                    </FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='token_reset_anchor'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Token reset anchor')}</FormLabel>
                    <FormControl>
                      <DateTimePicker
                        value={field.value}
                        onChange={(date) => {
                          field.onChange(date)
                          form.setValue(
                            'token_reset_timezone',
                            date
                              ? Intl.DateTimeFormat().resolvedOptions().timeZone
                              : ''
                          )
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Leave empty to inherit the plan schedule.')}
                    </FormDescription>
                  </FormItem>
                )}
              />
            </div>
          </form>
        </Form>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Cancel')}
          </SheetClose>
          <Button
            form='channel-pool-form'
            type='submit'
            disabled={form.formState.isSubmitting}
          >
            {form.formState.isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
