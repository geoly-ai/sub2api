<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">站内信</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">查看账户通知和风控提醒。</p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadMessages">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>

      <div class="card">
        <div v-if="loading" class="flex justify-center py-16">
          <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
        </div>
        <div v-else-if="messages.length === 0" class="px-6 py-16 text-center text-sm text-gray-500 dark:text-gray-400">
          暂无站内信
        </div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <div v-for="message in messages" :key="message.id" class="p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ message.title }}</h2>
                  <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="message.status === 'unread' ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">
                    {{ message.status === 'unread' ? '未读' : '已读' }}
                  </span>
                </div>
                <p class="mt-2 whitespace-pre-wrap text-sm leading-6 text-gray-600 dark:text-gray-300">{{ message.content }}</p>
                <p class="mt-3 text-xs text-gray-400">{{ formatDateTime(message.created_at) }}</p>
              </div>
              <button v-if="message.status === 'unread'" type="button" class="btn btn-secondary btn-sm" :disabled="markingID === message.id" @click="markRead(message.id)">
                标记已读
              </button>
            </div>
          </div>
        </div>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import messagesAPI, { type UserMessage } from '@/api/messages'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDateTimeValue } from '@/utils/format'

const appStore = useAppStore()
const loading = ref(false)
const markingID = ref<number | null>(null)
const messages = ref<UserMessage[]>([])
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 1 })

function formatDateTime(value?: string | null): string {
  return value ? formatDateTimeValue(value) : '-'
}

async function loadMessages() {
  loading.value = true
  try {
    const result = await messagesAPI.listMessages({
      page: pagination.page,
      page_size: pagination.page_size,
    })
    messages.value = result.items
    pagination.total = result.total
    pagination.page = result.page
    pagination.page_size = result.page_size
    pagination.pages = result.pages
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '加载站内信失败'))
  } finally {
    loading.value = false
  }
}

async function markRead(id: number) {
  markingID.value = id
  try {
    await messagesAPI.markMessageRead(id)
    messages.value = messages.value.map((item) => item.id === id ? { ...item, status: 'read', read_at: new Date().toISOString() } : item)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '标记已读失败'))
  } finally {
    markingID.value = null
  }
}

function onPageChange(page: number) {
  pagination.page = page
  void loadMessages()
}

function onPageSizeChange(pageSize: number) {
  pagination.page = 1
  pagination.page_size = pageSize
  void loadMessages()
}

onMounted(loadMessages)
</script>
