<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { instancesApi, ApiError, type Instance } from '@/lib/api'
import PhaseBadge from '@/components/PhaseBadge.vue'
import ExternalBadge from '@/components/ExternalBadge.vue'
import DatabaseIcon from '@/components/DatabaseIcon.vue'

const router = useRouter()

const instances = ref<Instance[]>([])
const loading = ref(true)
const loadError = ref('')

const deletingId = ref<string | null>(null)

function openInstance(id: string) {
  router.push(`/instances/${id}`)
}

type SortField = 'health' | 'access' | 'size'
const sortField = ref<SortField>('health')
const sortDesc = ref(false)

// Degraded first (needs attention), then Provisioning, then settled Healthy.
const healthOrder: Record<string, number> = { Degraded: 0, Provisioning: 1, Healthy: 2, Deleting: 3 }

// Kubernetes quantity notation ("5Gi", "500Mi", "1Ti", ...) doesn't sort
// correctly as a string ("10Gi" < "5Gi" alphabetically), so parse to bytes.
function parseStorageSize(size?: string): number {
  if (!size) return 0
  const match = size.match(/^(\d+(?:\.\d+)?)\s*([EPTGMK]i?)?B?$/i)
  if (!match) return 0
  const value = parseFloat(match[1] ?? '0')
  const unit = (match[2] ?? '').toLowerCase()
  const multipliers: Record<string, number> = {
    '': 1,
    k: 1000,
    m: 1000 ** 2,
    g: 1000 ** 3,
    t: 1000 ** 4,
    p: 1000 ** 5,
    e: 1000 ** 6,
    ki: 1024,
    mi: 1024 ** 2,
    gi: 1024 ** 3,
    ti: 1024 ** 4,
    pi: 1024 ** 5,
    ei: 1024 ** 6,
  }
  return value * (multipliers[unit] ?? 1)
}

const sortedInstances = computed(() => {
  const list = [...instances.value]
  list.sort((a, b) => {
    let cmp = 0
    switch (sortField.value) {
      case 'health':
        cmp = (healthOrder[a.phase] ?? 99) - (healthOrder[b.phase] ?? 99)
        break
      case 'access':
        cmp = Number(!!b.external) - Number(!!a.external) // external first
        break
      case 'size':
        cmp = parseStorageSize(a.storageSize) - parseStorageSize(b.storageSize)
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return list
})

let pollHandle: ReturnType<typeof setInterval> | undefined

async function refresh() {
  try {
    instances.value = await instancesApi.list()
    loadError.value = ''
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to load instances.'
  } finally {
    loading.value = false
  }
}

async function onDelete(id: string) {
  if (!confirm(`Delete instance "${id}"? This cannot be undone.`)) return
  deletingId.value = id
  try {
    await instancesApi.remove(id)
    await refresh()
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to delete instance.'
  } finally {
    deletingId.value = null
  }
}

onMounted(() => {
  refresh()
  pollHandle = setInterval(refresh, 5000)
})

onUnmounted(() => {
  clearInterval(pollHandle)
})
</script>

<template>
  <div class="mx-auto max-w-6xl px-6 py-10 lg:px-12">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-foreground">Instances</h1>
      <RouterLink
        to="/instances/new"
        class="flex items-center gap-1.5 rounded-md bg-accent px-4 py-2.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong"
      >
        <DatabaseIcon :size="14" />
        New instance
      </RouterLink>
    </div>

    <p v-if="loadError" class="mb-4 text-sm text-danger">{{ loadError }}</p>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>

    <div v-else-if="sortedInstances.length === 0" class="rounded-lg border border-dashed border-border-strong px-6 py-12 text-center text-sm text-muted">
      No instances yet. Create one to get started.
    </div>

    <template v-else>
      <div class="mb-4 flex items-center gap-2.5">
        <label for="sortField" class="text-sm text-faint">Sort by</label>
        <select
          id="sortField"
          v-model="sortField"
          class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground focus:border-accent focus:outline-none"
        >
          <option value="health">Health</option>
          <option value="access">Access</option>
          <option value="size">Size</option>
        </select>
        <button
          type="button"
          class="rounded-md border border-border bg-surface p-2 text-faint hover:text-foreground"
          :title="sortDesc ? 'Descending' : 'Ascending'"
          @click="sortDesc = !sortDesc"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="transition-transform"
            :class="sortDesc ? 'rotate-180' : ''"
          >
            <path d="M12 19V5M6 13l6 6 6-6" />
          </svg>
        </button>
      </div>

      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="instance in sortedInstances"
          :key="instance.id"
          class="group flex cursor-pointer flex-col gap-4 rounded-lg border border-border bg-surface-2 p-5 shadow-sm transition-colors hover:border-accent"
          @click="openInstance(instance.id)"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="flex min-w-0 items-center gap-2">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
                <DatabaseIcon :size="14" />
              </span>
              <span class="min-w-0 truncate font-mono text-base font-semibold text-foreground group-hover:text-accent">
                {{ instance.name }}
              </span>
            </div>
            <button
              :disabled="deletingId === instance.id"
              class="shrink-0 rounded-md p-2 text-faint hover:bg-danger-soft hover:text-danger disabled:opacity-50"
              :title="deletingId === instance.id ? 'Deleting…' : 'Delete instance'"
              @click.stop="onDelete(instance.id)"
            >
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13" />
              </svg>
            </button>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <PhaseBadge :phase="instance.phase" />
            <ExternalBadge :external="instance.external" />
          </div>

          <dl class="flex flex-col gap-2 border-t border-border pt-4 text-sm">
            <div class="flex items-center justify-between">
              <dt class="text-faint">Pods</dt>
              <dd class="text-foreground">{{ instance.readyInstances }} / {{ instance.instances }}</dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-faint">Size</dt>
              <dd class="font-mono text-foreground">{{ instance.storageSize || '—' }}</dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-faint">Created</dt>
              <dd class="text-foreground">{{ new Date(instance.createdAt).toLocaleDateString() }}</dd>
            </div>
          </dl>
        </div>
      </div>
    </template>
  </div>
</template>
