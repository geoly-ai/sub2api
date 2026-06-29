import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import McpView from '../McpView.vue'

const copyToClipboard = vi.hoisted(() => vi.fn())

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))

describe('McpView', () => {
  beforeEach(() => {
    copyToClipboard.mockReset()
  })

  it('renders endpoint and copyable generic HTTP configs', async () => {
    const wrapper = mount(McpView)

    expect(wrapper.text()).toContain('管理员 MCP')
    expect(wrapper.text()).toContain('http://localhost:3000/api/v1/admin/mcp')
    expect(wrapper.text()).toContain('<SUB2API_ADMIN_API_KEY>')
    expect(wrapper.text()).toContain('<SUB2API_ADMIN_JWT>')

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons[1].trigger('click')
    await buttons[2].trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith('http://localhost:3000/api/v1/admin/mcp', 'MCP 端点已复制')
    expect(copyToClipboard.mock.calls[1][0]).toContain('"x-api-key": "<SUB2API_ADMIN_API_KEY>"')
    expect(copyToClipboard.mock.calls[2][0]).toContain('"Authorization": "Bearer <SUB2API_ADMIN_JWT>"')
  })
})
