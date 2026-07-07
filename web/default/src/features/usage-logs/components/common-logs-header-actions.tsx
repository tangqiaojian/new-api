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
import { Eye, EyeOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { CommonLogsStats } from './common-logs-stats'
import { useUsageLogsContext } from './usage-logs-provider'

/**
 * Page-header actions for the Common Logs view: live usage stats plus a
 * toggle for masking sensitive values (token names, usernames, group names,
 * and the quota figure shown in stats), and a toggle for including cache-read
 * tokens in the displayed totals. Both controls live in the page header so
 * the toolbar below stays focused on filter inputs and form actions only.
 */
export function CommonLogsHeaderActions() {
  const { t } = useTranslation()
  const {
    sensitiveVisible,
    setSensitiveVisible,
    includeCache,
    setIncludeCache,
  } = useUsageLogsContext()

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <CommonLogsStats />
      <div className='flex items-center gap-1.5'>
        <Switch
          checked={includeCache}
          onCheckedChange={setIncludeCache}
          id='usage-logs-include-cache-switch'
        />
        <Label
          htmlFor='usage-logs-include-cache-switch'
          className='text-muted-foreground cursor-pointer text-xs font-normal'
        >
          {t('Include cache')}
        </Label>
      </div>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              onClick={() => setSensitiveVisible(!sensitiveVisible)}
              aria-label={sensitiveVisible ? t('Hide') : t('Show')}
              className='text-muted-foreground hover:text-foreground size-7'
            />
          }
        >
          {sensitiveVisible ? <Eye /> : <EyeOff />}
        </TooltipTrigger>
        <TooltipContent>
          {sensitiveVisible ? t('Hide') : t('Show')}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
