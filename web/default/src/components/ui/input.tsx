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
import { Input as InputPrimitive } from '@base-ui/react/input'
import * as React from 'react'

import { cn } from '@/lib/utils'

const inputBaseClassName =
  'border-input file:text-foreground placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 disabled:bg-input/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:bg-input/30 dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 h-8 w-full min-w-0 rounded-lg border bg-transparent px-2.5 py-1 text-base transition-colors outline-none file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:ring-3 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:ring-3 md:text-sm'

type InputProps = React.ComponentProps<'input'> & {
  suffix?: React.ReactNode
}

function Input({ className, type, suffix, ...props }: InputProps) {
  if (!suffix) {
    return (
      <InputPrimitive
        type={type}
        data-slot='input'
        className={cn(inputBaseClassName, className)}
        {...props}
      />
    )
  }

  // FormControl may merge a11y/id attrs onto this component. Keep them on the
  // real <input>, and put layout width classes on the wrapper so flex-1 works.
  const {
    id,
    disabled,
    'aria-describedby': ariaDescribedBy,
    'aria-invalid': ariaInvalid,
    'aria-labelledby': ariaLabelledBy,
    ...rest
  } = props

  return (
    <div className={cn('flex w-full min-w-0', className)} data-slot='input-group'>
      <InputPrimitive
        type={type}
        data-slot='input'
        id={id}
        disabled={disabled}
        aria-describedby={ariaDescribedBy}
        aria-invalid={ariaInvalid}
        aria-labelledby={ariaLabelledBy}
        className={cn(inputBaseClassName, 'min-w-0 flex-1 rounded-r-none pr-0')}
        {...rest}
      />
      <span
        className={cn(
          'text-muted-foreground border-input flex h-8 shrink-0 items-center rounded-r-lg border border-l-0 bg-transparent px-2 text-xs font-medium',
          disabled && 'pointer-events-none opacity-50'
        )}
      >
        {suffix}
      </span>
    </div>
  )
}

export { Input }
