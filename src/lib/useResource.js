import { useCallback, useEffect, useRef, useState } from 'react'

import { ApiError } from './api'
import { t } from './i18n'

/**
 * Turn an API error into copy a user can act on.
 *
 * Backend messages are safe to display — they are written for clients and
 * never quote PHI or internals — but the common cases read better as our own
 * strings, and a 500's opaque message is useless to anyone.
 */
export function describeError(error) {
  if (!(error instanceof ApiError)) return t('common.somethingWentWrong')

  switch (error.status) {
    case 0:
      return t('errors.network')
    case 401:
      return t('common.sessionExpired')
    case 403:
      return t('errors.forbidden')
    case 404:
      return t('errors.notFound')
    case 429:
      return t('login.rateLimited')
    case 500:
      return t('common.somethingWentWrong')
    default:
      return error.message || t('common.somethingWentWrong')
  }
}

/**
 * Load data from the API with loading, error, and refetch handling.
 *
 * `fetcher` receives an AbortSignal and must pass it through, so a request in
 * flight when the user navigates away is cancelled rather than resolving into
 * an unmounted component.
 *
 * Loading is *derived*, not stored: each request has a key, and the hook is
 * loading whenever the settled result's key is not the current one. Setting a
 * loading flag synchronously inside the effect would work too, but it costs an
 * extra render pass on every dependency change and React's lint rules
 * (correctly) flag it.
 */
export function useResource(fetcher, dependencies = []) {
  const [reloadToken, setReloadToken] = useState(0)
  const reload = useCallback(() => setReloadToken((token) => token + 1), [])

  // Identity of the request the caller currently wants. Serialised so a fresh
  // array literal on every render does not read as a change.
  const requestKey = JSON.stringify([dependencies, reloadToken])

  const [settled, setSettled] = useState({ key: null, data: null, error: null })

  // The fetcher closure is usually redefined every render. Holding it in a ref
  // — updated in its own effect, which runs before the fetch effect below
  // because effects fire in declaration order — keeps requestKey the single
  // trigger for refetching.
  const fetcherRef = useRef(fetcher)
  useEffect(() => {
    fetcherRef.current = fetcher
  })

  useEffect(() => {
    const controller = new AbortController()
    let cancelled = false

    fetcherRef
      .current(controller.signal)
      .then((data) => {
        if (!cancelled) setSettled({ key: requestKey, data, error: null })
      })
      .catch((cause) => {
        if (cancelled || cause?.name === 'AbortError') return
        setSettled({ key: requestKey, data: null, error: cause })
      })

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [requestKey])

  const loading = settled.key !== requestKey

  return {
    data: loading ? null : settled.data,
    error: loading ? null : settled.error,
    loading,
    reload,
  }
}
