<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Key 异常风控</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            查看 API Key 泄露疑似事件、异常调用规则、证据和封禁状态。
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadEvents">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>

      <div class="card">
        <div class="space-y-4 p-6">
          <div
            v-if="unavailableMessage"
            class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100"
          >
            {{ unavailableMessage }}
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-4">
            <input v-model.trim="filters.search" type="search" class="input" placeholder="搜索用户邮箱、Key 名称、规则" @keyup.enter="reloadFromFirstPage" />
            <Select v-model="filters.status" :options="statusOptions" @change="reloadFromFirstPage" />
            <Select v-model="filters.severity" :options="severityOptions" @change="reloadFromFirstPage" />
            <Select v-model="filters.rule_code" :options="ruleOptions" @change="reloadFromFirstPage" />
          </div>

          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">时间</th>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">用户 / Key</th>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">规则</th>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">证据</th>
                  <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">状态</th>
                  <th class="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
                <tr v-if="events.length === 0">
                  <td colspan="6" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                    {{ unavailableMessage || '暂无 API Key 异常事件' }}
                  </td>
                </tr>
                <tr v-for="event in events" :key="event.id">
                  <td class="whitespace-nowrap px-5 py-4 text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(event.created_at) }}</td>
                  <td class="px-5 py-4">
                    <p class="text-sm font-medium text-gray-900 dark:text-white">{{ event.user_email || `#${event.user_id}` }}</p>
                    <p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{{ event.api_key_name || '-' }} · #{{ event.api_key_id }}</p>
                  </td>
                  <td class="px-5 py-4">
                    <p class="font-mono text-sm text-gray-900 dark:text-white">{{ event.rule_code }}</p>
                    <p class="mt-1 text-xs" :class="event.severity === 'high' ? 'text-rose-600 dark:text-rose-300' : 'text-amber-600 dark:text-amber-300'">
                      {{ event.severity }} · {{ event.score }}
                    </p>
                  </td>
                  <td class="max-w-md px-5 py-4 text-xs text-gray-600 dark:text-gray-300">
                    <p class="truncate">{{ evidenceSummary(event) }}</p>
                    <p class="mt-1 truncate font-mono text-gray-400">{{ evidenceSamples(event) }}</p>
                  </td>
                  <td class="px-5 py-4">
                    <span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="statusClass(event.status)">
                      {{ event.status }}
                    </span>
                  </td>
                  <td class="px-5 py-4 text-right">
                    <div class="flex justify-end gap-2">
                      <button type="button" class="btn btn-secondary btn-sm" :disabled="actionID === event.id" @click="resolveEvent(event)">处理</button>
                      <button v-if="event.status === 'blocked'" type="button" class="btn btn-primary btn-sm" :disabled="actionID === event.id" @click="unblockEvent(event)">解封 Key</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="onPageChange"
            @update:page-size="onPageSizeChange"
          />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import { adminAPI } from '@/api/admin'
import type { APIKeyRiskEvent } from '@/api/admin/riskControl'
import type { SelectOption } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDateTimeValue } from '@/utils/format'

const appStore = useAppStore()
const loading = ref(false)
const actionID = ref<number | null>(null)
const unavailableMessage = ref('')
const events = ref<APIKeyRiskEvent[]>([])

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 1,
})

const filters = reactive({
  search: '',
  status: '',
  severity: '',
  rule_code: '',
})

const statusOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部状态' },
  { value: 'open', label: '待处理' },
  { value: 'blocked', label: '已封禁' },
  { value: 'resolved', label: '已处理' },
])

const severityOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部级别' },
  { value: 'high', label: '高危' },
  { value: 'medium', label: '中危' },
])

const ruleOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部规则' },
  { value: 'key_multi_ip_30m', label: 'Key 多 IP' },
  { value: 'user_multi_ip_30m', label: '用户多 IP' },
  { value: 'off_hours_spike', label: '夜间突增' },
  { value: 'ua_ip_churn_60m', label: 'UA/IP 变化' },
])

async function loadEvents() {
  loading.value = true
  try {
    const result = await adminAPI.riskControl.listAPIKeyRiskEvents({
      page: pagination.page,
      page_size: pagination.page_size,
      search: filters.search || undefined,
      status: filters.status || undefined,
      severity: filters.severity || undefined,
      rule_code: filters.rule_code || undefined,
    })
    events.value = result.items
    pagination.total = result.total
    pagination.page = result.page
    pagination.page_size = result.page_size
    pagination.pages = result.pages
    unavailableMessage.value = ''
  } catch (err: unknown) {
    if (isEventsUnavailable(err)) {
      events.value = []
      pagination.total = 0
      pagination.pages = 1
      unavailableMessage.value = 'API Key 异常事件接口暂不可用，请确认后端已发布并完成数据库迁移。'
      return
    }
    unavailableMessage.value = ''
    appStore.showError(extractApiErrorMessage(err, '加载 API Key 异常事件失败'))
  } finally {
    loading.value = false
  }
}

function isEventsUnavailable(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false
  const e = err as { status?: number; message?: string; error?: string; response?: { status?: number; data?: { message?: string; error?: string } } }
  const status = e.status ?? e.response?.status
  if (status === 404) return true
  const message = `${e.message || ''} ${e.error || ''} ${e.response?.data?.message || ''} ${e.response?.data?.error || ''}`.toLowerCase()
  return (
    message.includes('api_key_risk_events') ||
    (message.includes('relation') && message.includes('does not exist')) ||
    (message.includes('column') && message.includes('risk_blocked'))
  )
}

function reloadFromFirstPage() {
  pagination.page = 1
  void loadEvents()
}

function onPageChange(page: number) {
  pagination.page = page
  void loadEvents()
}

function onPageSizeChange(pageSize: number) {
  pagination.page = 1
  pagination.page_size = pageSize
  void loadEvents()
}

async function resolveEvent(event: APIKeyRiskEvent) {
  if (actionID.value !== null) return
  actionID.value = event.id
  try {
    await adminAPI.riskControl.resolveAPIKeyRiskEvent(event.id)
    await loadEvents()
    appStore.showSuccess('已标记处理')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '处理 API Key 异常事件失败'))
  } finally {
    actionID.value = null
  }
}

async function unblockEvent(event: APIKeyRiskEvent) {
  if (actionID.value !== null) return
  actionID.value = event.id
  try {
    await adminAPI.riskControl.unblockAPIKeyRisk(event.api_key_id)
    await adminAPI.riskControl.resolveAPIKeyRiskEvent(event.id)
    await loadEvents()
    appStore.showSuccess('Key 已解封')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '解封 API Key 失败'))
  } finally {
    actionID.value = null
  }
}

function statusClass(status: string): string {
  if (status === 'blocked') return 'bg-rose-50 text-rose-700 dark:bg-rose-900/20 dark:text-rose-300'
  if (status === 'resolved') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
}

function evidenceSummary(event: APIKeyRiskEvent): string {
  const ev = event.evidence || {}
  const parts = [
    ev.requests !== undefined ? `请求 ${ev.requests}` : '',
    ev.user_requests !== undefined ? `用户请求 ${ev.user_requests}` : '',
    ev.off_hours_requests !== undefined ? `夜间请求 ${ev.off_hours_requests}` : '',
    ev.ip_count !== undefined ? `IP ${ev.ip_count}` : '',
    ev.user_ip_count !== undefined ? `用户 IP ${ev.user_ip_count}` : '',
    ev.ua_family_count !== undefined ? `UA ${ev.ua_family_count}` : '',
  ].filter(Boolean)
  return parts.join(' · ') || String(ev.threshold || '-')
}

function evidenceSamples(event: APIKeyRiskEvent): string {
  const ev = event.evidence || {}
  const ips = Array.isArray(ev.sample_ips) ? ev.sample_ips.join(', ') : ''
  const uas = Array.isArray(ev.sample_user_agents) ? ev.sample_user_agents.join(', ') : ''
  return [ips, uas].filter(Boolean).join(' · ') || '-'
}

function formatDateTime(value?: string | null): string {
  return formatDateTimeValue(value || '')
}

onMounted(() => {
  void loadEvents()
})
</script>
