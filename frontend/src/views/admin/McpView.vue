<template>
  <div class="space-y-6">
    <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">管理员 MCP</h2>
          <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
            该端点仅接受管理员认证，可供支持 MCP HTTP 连接的客户端调用用户查询、批量创建、批量加余额和批量禁用工具。
          </p>
        </div>
        <button class="btn btn-primary" type="button" @click="copyEndpoint">
          复制端点
        </button>
      </div>

      <div class="mt-5 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-950">
        <div class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">MCP HTTP URL</div>
        <code class="break-all font-mono text-sm text-gray-900 dark:text-white">{{ endpointUrl }}</code>
      </div>
    </section>

    <section class="grid gap-6 xl:grid-cols-2">
      <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">Admin API Key 配置</h3>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">推荐用于长期部署。请将占位符替换为系统设置中的管理员 API Key。</p>
          </div>
          <button class="btn btn-secondary flex-shrink-0" type="button" @click="copyAPIKeyConfig">
            复制配置
          </button>
        </div>
        <pre class="config-block">{{ apiKeyConfig }}</pre>
      </div>

      <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">管理员 JWT 配置</h3>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">适合临时连接。JWT 过期或管理员改密后需要重新复制。</p>
          </div>
          <button class="btn btn-secondary flex-shrink-0" type="button" @click="copyJWTConfig">
            复制配置
          </button>
        </div>
        <pre class="config-block">{{ jwtConfig }}</pre>
      </div>
    </section>

    <section class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100">
      管理员 MCP 拥有批量修改用户的能力。请只把配置交给可信客户端，避免在聊天、工单或日志中粘贴真实密钥。
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { buildApiUrl } from '@/api/client'
import { useClipboard } from '@/composables/useClipboard'

const { copyToClipboard } = useClipboard()

const endpointUrl = computed(() => {
  const apiPath = buildApiUrl('/admin/mcp')
  try {
    return new URL(apiPath, window.location.origin).href
  } catch {
    return apiPath
  }
})

const apiKeyConfig = computed(() => JSON.stringify({
  name: 'sub2api-admin',
  type: 'http',
  url: endpointUrl.value,
  headers: {
    'x-api-key': '<SUB2API_ADMIN_API_KEY>'
  }
}, null, 2))

const jwtConfig = computed(() => JSON.stringify({
  name: 'sub2api-admin',
  type: 'http',
  url: endpointUrl.value,
  headers: {
    Authorization: 'Bearer <SUB2API_ADMIN_JWT>'
  }
}, null, 2))

function copyEndpoint() {
  void copyToClipboard(endpointUrl.value, 'MCP 端点已复制')
}

function copyAPIKeyConfig() {
  void copyToClipboard(apiKeyConfig.value, 'Admin API Key 配置已复制')
}

function copyJWTConfig() {
  void copyToClipboard(jwtConfig.value, '管理员 JWT 配置已复制')
}
</script>

<style scoped>
.config-block {
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  border-radius: 0.5rem;
  border: 1px solid rgb(229 231 235);
  background: rgb(249 250 251);
  padding: 1rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.875rem;
  line-height: 1.5;
  color: rgb(17 24 39);
}

:global(.dark) .config-block {
  border-color: rgb(55 65 81);
  background: rgb(3 7 18);
  color: rgb(243 244 246);
}
</style>
