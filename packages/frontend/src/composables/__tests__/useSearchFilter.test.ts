import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref } from 'vue'
import { useSearchFilter } from '../useSearchFilter'

// Mock @vueuse/core
const mockStorageValue = ref<{ tags: string[]; date: string } | null>(null)
vi.mock('@vueuse/core', () => ({
  useStorage: vi.fn(() => mockStorageValue),
  useDebounceFn: vi.fn((fn) => fn),
}))

// Mock contentApi
vi.mock('@/api', () => ({
  contentApi: {
    getTags: vi.fn(),
  },
}))

// Mock useHomeStore
vi.mock('@/stores/home', () => ({
  useHomeStore: vi.fn(() => ({
    selectedTags: [],
    searchKeyword: '',
  })),
}))

describe('useSearchFilter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockStorageValue.value = null
  })

  afterEach(() => vi.restoreAllMocks())

  it('should initialize with default values', () => {
    const { allTags, selectedTags, searchKeyword } = useSearchFilter()

    expect(allTags.value).toEqual([])
    expect(selectedTags.value).toEqual([])
    expect(searchKeyword.value).toBe('')
  })

  it('should select and deselect tags', () => {
    const { selectedTags, selectTag } = useSearchFilter()
    const callback = vi.fn()

    selectTag('tag1', callback)
    expect(selectedTags.value).toEqual(['tag1'])
    expect(callback).toHaveBeenCalled()

    selectTag('tag1', callback)
    expect(selectedTags.value).toEqual([])
  })

  it('should handle search with debounce', () => {
    const { handleSearch } = useSearchFilter()
    const callback = vi.fn()

    handleSearch(callback)
    expect(callback).toHaveBeenCalled()
  })

  it('should load tags from API', async () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    const { loadTags, allTags, shuffledTags } = useSearchFilter()
    const mockTags = ['tag1', 'tag2', 'tag3', 'tag4']

    const { contentApi } = await import('@/api')
    vi.mocked(contentApi.getTags).mockResolvedValue({
      code: 200,
      message: 'success',
      data: mockTags,
    })

    await loadTags()
    expect(allTags.value).toEqual(mockTags)
    expect(shuffledTags.value).not.toEqual(mockTags)
    expect([...shuffledTags.value].sort()).toEqual(mockTags)
    expect(mockStorageValue.value?.tags).toEqual(mockTags)
  })

  it('should use cached tags when available', async () => {
    const mockTags = ['cached1', 'cached2']

    // Set the mock storage value before using the composable
    mockStorageValue.value = {
      tags: mockTags,
      date: new Date().toDateString(),
    }

    const { loadTags, allTags } = useSearchFilter()

    await loadTags()
    expect(allTags.value).toEqual(mockTags)
  })

  it('should shuffle daily cached tags for each new instance without changing the cache', async () => {
    const random = vi.spyOn(Math, 'random').mockReturnValue(0)
    const tags = ['a', 'b', 'c', 'd']
    mockStorageValue.value = { tags, date: new Date().toDateString() }
    const first = useSearchFilter()
    await first.loadTags()

    random.mockReturnValue(0.999)
    const second = useSearchFilter()
    await second.loadTags()

    expect(first.shuffledTags.value).not.toEqual(second.shuffledTags.value)
    expect([...first.shuffledTags.value].sort()).toEqual(tags)
    expect([...second.shuffledTags.value].sort()).toEqual(tags)
    expect(mockStorageValue.value.tags).toEqual(['a', 'b', 'c', 'd'])
    const { contentApi } = await import('@/api')
    expect(contentApi.getTags).not.toHaveBeenCalled()
  })

  it('should preserve discovery order during filtering and reshuffle without fetching tags again', async () => {
    const random = vi.spyOn(Math, 'random').mockReturnValue(0)
    mockStorageValue.value = { tags: ['a', 'b', 'c', 'd'], date: new Date().toDateString() }
    const filter = useSearchFilter()
    await filter.loadTags()
    const initial = [...filter.shuffledTags.value]

    filter.selectTag('c', vi.fn())
    expect(filter.shuffledTags.value).toEqual(initial)
    filter.selectTag('c', vi.fn())
    expect(filter.shuffledTags.value).toEqual(initial)

    random.mockReturnValue(0.999)
    filter.reshuffleTags()
    expect(filter.shuffledTags.value).not.toEqual(initial)
    expect([...filter.shuffledTags.value].sort()).toEqual(['a', 'b', 'c', 'd'])
    const { contentApi } = await import('@/api')
    expect(contentApi.getTags).not.toHaveBeenCalled()
  })

  it('should handle API error gracefully', async () => {
    const { loadTags, allTags } = useSearchFilter()

    const { contentApi } = await import('@/api')
    vi.mocked(contentApi.getTags).mockRejectedValue(new Error('API Error'))

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    await loadTags()
    expect(allTags.value).toEqual([])
    expect(consoleSpy).toHaveBeenCalled()

    consoleSpy.mockRestore()
  })
})
