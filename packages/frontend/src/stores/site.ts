import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { siteApi } from '@/api'
import type { SiteConfig } from '@/types'
import { DEFAULT_SITE_CONFIG, DEFAULT_SITE_TEXTS, renderSiteText, siteTextLines } from '@/utils/siteText'

function cloneDefault(): SiteConfig {
  return {
    ...DEFAULT_SITE_CONFIG,
    friend_links: DEFAULT_SITE_CONFIG.friend_links.map((link) => ({ ...link })),
    texts: { ...DEFAULT_SITE_TEXTS },
  }
}

function mergeConfig(input: Partial<SiteConfig>): SiteConfig {
  return {
    ...cloneDefault(),
    ...input,
    friend_links: input.friend_links ?? DEFAULT_SITE_CONFIG.friend_links,
    texts: { ...DEFAULT_SITE_TEXTS, ...(input.texts ?? {}) },
  }
}

function upsertMeta(name: string, content: string) {
  if (typeof document === 'undefined') return
  const element = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
  if (!content) {
    element?.remove()
    return
  }
  if (element) {
    element.content = content
    return
  }
  const created = document.createElement('meta')
  created.name = name
  created.content = content
  document.head.append(created)
}

function upsertLink(rel: string, href: string) {
  if (typeof document === 'undefined' || !href) return
  let element = document.querySelector<HTMLLinkElement>(`link[rel="${rel}"]`)
  if (!element) {
    element = document.createElement('link')
    element.rel = rel
    document.head.append(element)
  }
  element.href = href
}

export const useSiteStore = defineStore('site', () => {
  const config = ref<SiteConfig>(cloneDefault())
  const loaded = ref(false)
  const loading = ref(false)
  const error = ref('')

  const baseTitle = computed(() => renderSiteText(config.value.browser_title || DEFAULT_SITE_CONFIG.browser_title, config.value))

  function text(key: string): string {
    const template = config.value.texts[key] ?? DEFAULT_SITE_TEXTS[key] ?? ''
    return renderSiteText(template, config.value)
  }

  function format(key: string, values: Record<string, string>): string {
    return Object.entries(values).reduce(
      (result, [name, value]) => result.replaceAll(`{${name}}`, value),
      text(key),
    )
  }

  function lines(key: string): string[] {
    const template = config.value.texts[key] ?? DEFAULT_SITE_TEXTS[key] ?? ''
    return siteTextLines(template, config.value)
  }

  function applyConfig(input: SiteConfig) {
    config.value = mergeConfig(input)
    loaded.value = true
  }

  function applyDocument() {
    if (typeof document === 'undefined') return
    document.title = baseTitle.value
    upsertMeta('description', config.value.description)
    upsertMeta('application-name', config.value.short_name)
    upsertMeta('apple-mobile-web-app-title', config.value.short_name)
    upsertLink('icon', config.value.favicon_url || '/favicon.webp')
    upsertLink('apple-touch-icon', config.value.favicon_url || '/favicon.webp')
    upsertLink('manifest', '/api/site/manifest')
  }

  let pending: Promise<void> | null = null
  async function load(force = false) {
    if (loaded.value && !force) return
    if (pending) return pending
    pending = (async () => {
      loading.value = true
      error.value = ''
      try {
        const response = await siteApi.config()
        if (response.code === 200 && response.data) applyConfig(response.data)
      } catch (e) {
        error.value = (e as Error).message || '站点配置加载失败'
      } finally {
        loading.value = false
        loaded.value = true
        applyDocument()
        pending = null
      }
    })()
    return pending
  }

  watch(
    () => [
      config.value.name,
      config.value.short_name,
      config.value.tagline,
      config.value.browser_title,
      config.value.description,
      config.value.favicon_url,
    ],
    () => applyDocument(),
  )

  return { config, loaded, loading, error, baseTitle, text, format, lines, load, applyConfig, applyDocument }
})
