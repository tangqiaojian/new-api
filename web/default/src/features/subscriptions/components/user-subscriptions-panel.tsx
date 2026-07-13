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
import { useCallback, useState } from 'react'

import { AddQuotaDialog } from './dialogs/add-quota-dialog'
import { ResetSubscriptionConfirm } from './dialogs/reset-subscription-confirm'
import { UserSubscriptionsTable } from './user-subscriptions-table'
import type { AdminUserSubscriptionItem } from '../types'

/**
 * Wraps the user-subscriptions table with its own dialog state so the
 * add-quota / reset actions are self-contained within the tab.
 */
export function UserSubscriptionsPanel() {
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [addQuotaRow, setAddQuotaRow] =
    useState<AdminUserSubscriptionItem | null>(null)
  const [resetRow, setResetRow] =
    useState<AdminUserSubscriptionItem | null>(null)

  const triggerRefresh = useCallback(() => {
    setRefreshTrigger((prev) => prev + 1)
  }, [])

  return (
    <>
      <UserSubscriptionsTable
        onAddQuota={setAddQuotaRow}
        onReset={setResetRow}
        refreshTrigger={refreshTrigger}
      />
      <AddQuotaDialog
        open={addQuotaRow !== null}
        onOpenChange={(open) => !open && setAddQuotaRow(null)}
        subscription={addQuotaRow}
        onSuccess={triggerRefresh}
      />
      <ResetSubscriptionConfirm
        open={resetRow !== null}
        onOpenChange={(open) => !open && setResetRow(null)}
        subscription={resetRow}
        onSuccess={triggerRefresh}
      />
    </>
  )
}
