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
import assert from 'node:assert/strict'
import { after, afterEach, describe, it } from 'node:test'

import { Window } from 'happy-dom'

import {
  DASHBOARD_INCLUDE_CACHE_STORAGE_KEY,
  LEGACY_DASHBOARD_INCLUDE_CACHE_STORAGE_KEY,
} from '../../constants'
import { getSavedIncludeCache, saveIncludeCache } from '../filters'

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
const originalLocalStorage = Object.getOwnPropertyDescriptor(
  globalThis,
  'localStorage'
)
const domWindow = new Window()

Object.defineProperties(globalThis, {
  window: { configurable: true, value: domWindow },
  localStorage: { configurable: true, value: domWindow.localStorage },
})

afterEach(() => {
  domWindow.localStorage.clear()
})

after(() => {
  domWindow.close()
  if (originalWindow) {
    Object.defineProperty(globalThis, 'window', originalWindow)
  } else {
    Reflect.deleteProperty(globalThis, 'window')
  }
  if (originalLocalStorage) {
    Object.defineProperty(globalThis, 'localStorage', originalLocalStorage)
  } else {
    Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

describe('dashboard include-cache preference', () => {
  it('migrates the legacy preference to the versioned storage key', () => {
    domWindow.localStorage.setItem(
      LEGACY_DASHBOARD_INCLUDE_CACHE_STORAGE_KEY,
      'true'
    )

    assert.equal(getSavedIncludeCache(), true)
    assert.equal(
      domWindow.localStorage.getItem(DASHBOARD_INCLUDE_CACHE_STORAGE_KEY),
      'true'
    )
  })

  it('uses the versioned preference when both keys exist', () => {
    domWindow.localStorage.setItem(DASHBOARD_INCLUDE_CACHE_STORAGE_KEY, 'false')
    domWindow.localStorage.setItem(
      LEGACY_DASHBOARD_INCLUDE_CACHE_STORAGE_KEY,
      'true'
    )

    assert.equal(getSavedIncludeCache(), false)
  })

  it('persists changes under the versioned storage key', () => {
    saveIncludeCache(true)

    assert.equal(
      domWindow.localStorage.getItem(DASHBOARD_INCLUDE_CACHE_STORAGE_KEY),
      'true'
    )
  })
})
