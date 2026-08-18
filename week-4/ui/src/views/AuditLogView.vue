<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { auditLogsApi, ApiError, type AuditLogEntry } from '@/lib/api'

const PAGE_SIZE = 50

const entries = ref<AuditLogEntry[]>([])
const loading = ref(true)
const loadingMore = ref(false)
const loadError = ref('')
const hasMore = ref(false)

function formatAction(action: string): string {
  const label = action.replace('.', ' ')
  return label.charAt(0).toUpperCase() + label.slice(1)
}

function formatMetadata(metadata?: Record<string, unknown>): string {
  if (!metadata || Object.keys(metadata).length === 0) return '—'
  return Object.entries(metadata)
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(', ') || 'none' : value}`)
    .join(' · ')
}

async function load(offset: number) {
  const page = await auditLogsApi.list({ limit: PAGE_SIZE, offset })
  entries.value = offset === 0 ? page : [...entries.value, ...page]
  hasMore.value = page.length === PAGE_SIZE
}

async function loadMore() {
  loadingMore.value = true
  try {
    await load(entries.value.length)
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to load more activity.'
  } finally {
    loadingMore.value = false
  }
}

onMounted(async () => {
  try {
    await load(0)
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to load activity.'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 py-10 lg:px-12">
    <h1 class="mb-1 text-2xl font-semibold text-foreground">Activity</h1>
    <p class="mb-6 text-sm text-muted">
      A permanent record of actions on your account — instance creation, changes, deletion,
      and credential access.
    </p>

    <p v-if="loadError" class="mb-4 text-sm text-danger">{{ loadError }}</p>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>

    <div
      v-else-if="entries.length === 0"
      class="rounded-lg border border-dashed border-border-strong px-6 py-12 text-center text-sm text-muted"
    >
      No activity recorded yet.
    </div>

    <template v-else>
      <div class="overflow-x-auto rounded-lg border border-border bg-surface-2 shadow-sm">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-border text-xs uppercase tracking-wide text-faint">
              <th class="px-4 py-3 font-medium">When</th>
              <th class="px-4 py-3 font-medium">Action</th>
              <th class="px-4 py-3 font-medium">Instance</th>
              <th class="px-4 py-3 font-medium">Details</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in entries" :key="entry.id" class="border-b border-border last:border-0">
              <td class="whitespace-nowrap px-4 py-2.5 text-muted">{{ new Date(entry.createdAt).toLocaleString() }}</td>
              <td class="px-4 py-2.5 font-medium text-foreground">{{ formatAction(entry.action) }}</td>
              <td class="px-4 py-2.5 font-mono text-foreground">
                <RouterLink
                  v-if="entry.instanceName"
                  :to="`/instances/${entry.instanceName}`"
                  class="text-accent hover:underline"
                >
                  {{ entry.instanceName }}
                </RouterLink>
                <span v-else class="text-faint">—</span>
              </td>
              <td class="px-4 py-2.5 text-xs text-muted">{{ formatMetadata(entry.metadata) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="hasMore" class="mt-4 flex justify-center">
        <button
          :disabled="loadingMore"
          class="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-surface-hover disabled:opacity-50"
          @click="loadMore"
        >
          {{ loadingMore ? 'Loading…' : 'Load more' }}
        </button>
      </div>
    </template>
  </div>
</template>
