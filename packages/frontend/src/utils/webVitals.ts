
interface WebVitalMetric {
  name: string
  value: number
  rating: 'good' | 'needs-improvement' | 'poor'
  delta: number
  id: string
}

type MetricCallback = (metric: WebVitalMetric) => void

const thresholds = {
  LCP: { good: 2500, poor: 4000 },
  FID: { good: 100, poor: 300 },
  CLS: { good: 0.1, poor: 0.25 },
  TTFB: { good: 800, poor: 1800 },
  FCP: { good: 1800, poor: 3000 },
}

function getRating(name: string, value: number): 'good' | 'needs-improvement' | 'poor' {
  const threshold = thresholds[name as keyof typeof thresholds]
  if (!threshold) return 'good'

  if (value <= threshold.good) return 'good'
  if (value <= threshold.poor) return 'needs-improvement'
  return 'poor'
}

function generateId(): string {
  return `${Date.now()}-${crypto.randomUUID()}`
}

function observeLCP(callback: MetricCallback): void {
  if (!('PerformanceObserver' in globalThis)) return

  try {
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries()
      const lastEntry = entries.at(-1) as (PerformanceEntry & { startTime: number }) | undefined

      if (lastEntry) {
        callback({
          name: 'LCP',
          value: lastEntry.startTime,
          rating: getRating('LCP', lastEntry.startTime),
          delta: lastEntry.startTime,
          id: generateId(),
        })
      }
    })

    observer.observe({ type: 'largest-contentful-paint', buffered: true })
  } catch (e) {
    console.warn('LCP observation failed:', e)
  }
}

function observeFID(callback: MetricCallback): void {
  if (!('PerformanceObserver' in globalThis)) return

  try {
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries()
      entries.forEach((entry) => {
        const fidEntry = entry as PerformanceEntry & { processingStart: number; startTime: number }
        const fid = fidEntry.processingStart - fidEntry.startTime

        callback({
          name: 'FID',
          value: fid,
          rating: getRating('FID', fid),
          delta: fid,
          id: generateId(),
        })
      })
    })

    observer.observe({ type: 'first-input', buffered: true })
  } catch (e) {
    console.warn('FID observation failed:', e)
  }
}

function observeCLS(callback: MetricCallback): void {
  if (!('PerformanceObserver' in globalThis)) return

  let clsValue = 0

  try {
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries()
      entries.forEach((entry) => {
        const clsEntry = entry as PerformanceEntry & { hadRecentInput: boolean; value: number }
        if (!clsEntry.hadRecentInput) {
          clsValue += clsEntry.value
        }
      })

      callback({
        name: 'CLS',
        value: clsValue,
        rating: getRating('CLS', clsValue),
        delta: clsValue,
        id: generateId(),
      })
    })

    observer.observe({ type: 'layout-shift', buffered: true })
  } catch (e) {
    console.warn('CLS observation failed:', e)
  }
}

function observeTTFB(callback: MetricCallback): void {
  if (!('performance' in globalThis) || !('timing' in performance)) return

  const navEntry = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming

  if (navEntry) {
    const ttfb = navEntry.responseStart - navEntry.requestStart

    callback({
      name: 'TTFB',
      value: ttfb,
      rating: getRating('TTFB', ttfb),
      delta: ttfb,
      id: generateId(),
    })
  }
}

function observeFCP(callback: MetricCallback): void {
  if (!('PerformanceObserver' in globalThis)) return

  try {
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries()
      entries.forEach((entry) => {
        if (entry.name === 'first-contentful-paint') {
          callback({
            name: 'FCP',
            value: entry.startTime,
            rating: getRating('FCP', entry.startTime),
            delta: entry.startTime,
            id: generateId(),
          })
        }
      })
    })

    observer.observe({ type: 'paint', buffered: true })
  } catch (e) {
    console.warn('FCP observation failed:', e)
  }
}

export function initWebVitals(callback?: MetricCallback): void {
  const defaultCallback = (metric: WebVitalMetric) => {
    if (!import.meta.env.DEV) return
    let emoji = '❌'
    if (metric.rating === 'good') {
      emoji = '✅'
    } else if (metric.rating === 'needs-improvement') {
      emoji = '⚠️'
    }
    console.log(`${emoji} [Web Vitals] ${metric.name}: ${metric.value.toFixed(2)}ms (${metric.rating})`)
  }

  const metricCallback = callback || defaultCallback

  if (document.readyState === 'complete') {
    startObservers(metricCallback)
  } else {
    globalThis.addEventListener('load', () => {
      setTimeout(() => startObservers(metricCallback), 0)
    })
  }
}

function startObservers(callback: MetricCallback): void {
  observeLCP(callback)
  observeFID(callback)
  observeCLS(callback)
  observeTTFB(callback)
  observeFCP(callback)
}
