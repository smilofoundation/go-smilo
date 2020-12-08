import Deferred from 'src/utils/Deferred'

import {InfluxLanguage} from 'src/types/v2/dashboards'
import {DashboardQuery} from 'src/types/v2/dashboards'

const CHECK_LIMIT_INTERVAL = 200
const MAX_ROWS = 50000

interface ExecuteFluxQueryResult {
  csv: string
  didTruncate: boolean
  rowCount: number
}

interface XHRError extends Error {
  xhr?: XMLHttpRequest
}

export const executeQuery = async (
  url: string,
  query: string,
  language: InfluxLanguage = InfluxLanguage.Flux
): Promise<ExecuteFluxQueryResult> => {
  // We're using `XMLHttpRequest` directly here rather than through `axios` so
  // that we can poll the response size as it comes back. If the response size
  // is greater than a predefined limit, we close the HTTP connection and
  // return the partial response. We could acheive this more elegantly using
  // `fetch` and the [Streams API][0], but the Streams API is currently behind
  // a feature flag in Firefox.
  //
  // [0]: https://developer.mozilla.org/en-US/docs/Web/API/Streams_API
  const xhr = new XMLHttpRequest()
  const deferred = new Deferred()

  let didTruncate = false
  let rowCount = 0
  let rowCountIndex = 0

  const countNewRows = (): number => {
    // Don't extract this to a non-closure helper, since passing
    // `xhr.responseText` as an argument will be expensive
    if (!xhr.responseText) {
      return 0
    }

    let count = 0

    for (let i = rowCountIndex; i < xhr.responseText.length; i++) {
      if (xhr.responseText[i] === '\n') {
        count++
      }
    }

    return count
  }

  const resolve = () => {
    let csv = xhr.responseText

    if (didTruncate) {
      // Discard the last line in the response since it may be partially read
      csv = csv.slice(0, csv.lastIndexOf('\n'))
    }

    rowCount += countNewRows()

    const result: ExecuteFluxQueryResult = {
      csv,
      didTruncate,
      rowCount,
    }

    deferred.resolve(result)
    clearTimeout(interval)
  }

  const reject = () => {
    let bodyError = null

    try {
      bodyError = JSON.parse(xhr.responseText).error
    } catch {
      if (xhr.responseText && xhr.responseText.trim() !== '') {
        bodyError = xhr.responseText
      }
    }

    const statusError = xhr.statusText
    const fallbackError = 'failed to execute Flux query'
    const error: XHRError = new Error(bodyError || statusError || fallbackError)
    error.xhr = xhr

    deferred.reject(error)
    clearTimeout(interval)
  }

  const interval = setInterval(() => {
    if (!xhr.responseText) {
      // Haven't received any data yet
      return
    }

    rowCount += countNewRows()
    rowCountIndex = xhr.responseText.length

    if (rowCount < MAX_ROWS) {
      return
    }

    didTruncate = true
    resolve()
    xhr.abort()
  }, CHECK_LIMIT_INTERVAL)

  xhr.onload = () => {
    if (xhr.status === 200) {
      resolve()
    } else {
      reject()
    }
  }

  xhr.onerror = reject

  const dialect = {annotations: ['group', 'datatype', 'default']}
  const body = JSON.stringify({query, dialect, type: language})

  xhr.open('POST', url)
  xhr.setRequestHeader('Content-Type', 'application/json')
  xhr.send(body)

  return deferred.promise
}

export const executeQueries = async (
  url: string,
  queries: DashboardQuery[]
): Promise<ExecuteFluxQueryResult[]> => {
  return Promise.all(
    queries.map(({type, text}) => executeQuery(url, text, type))
  )
}
