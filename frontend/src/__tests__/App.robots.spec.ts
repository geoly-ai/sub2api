import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import App from '@/App.vue'

vi.mock('@/api/setup', () => ({
  getSetupStatus: vi.fn(async () => ({ needs_setup: false }))
}))

vi.mock('@/router/title', () => ({
  resolveRouteDocumentTitle: () => 'Sub2API'
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteName: 'Sub2API',
    siteLogo: '',
    cachedPublicSettings: { custom_menu_items: [] },
    fetchPublicSettings: vi.fn(async () => undefined),
  }),
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false,
  }),
  useSubscriptionStore: () => ({
    fetchActiveSubscriptions: vi.fn(async () => undefined),
    startPolling: vi.fn(),
    clear: vi.fn(),
  }),
  useAnnouncementStore: () => ({
    fetchAnnouncements: vi.fn(),
    reset: vi.fn(),
  }),
  useAdminComplianceStore: () => ({
    fetchStatus: vi.fn(async () => undefined),
    requireAcknowledgement: vi.fn(),
    reset: vi.fn(),
  }),
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

describe('App robots meta', () => {
  beforeEach(() => {
    document.head.querySelectorAll('meta[name="robots"][data-route-managed="true"]').forEach((node) => node.remove())
  })

  it('sets noindex,nofollow on home and clears it on other routes', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/home', component: { template: '<div />' } },
        { path: '/dashboard', component: { template: '<div />' } },
      ],
    })
    await router.push('/home')
    await router.isReady()

    mount(App, {
      global: {
        plugins: [router],
        stubs: {
          NavigationProgress: true,
          Toast: true,
          AdminComplianceDialog: true,
          AnnouncementPopup: true,
        },
      },
    })

    await nextTick()
    expect(document.head.querySelector('meta[name="robots"]')?.getAttribute('content')).toBe('noindex,nofollow')

    await router.push('/dashboard')
    await nextTick()
    expect(document.head.querySelector('meta[name="robots"][data-route-managed="true"]')).toBeNull()
  })
})
