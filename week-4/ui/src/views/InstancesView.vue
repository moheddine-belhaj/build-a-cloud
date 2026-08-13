<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { instancesApi, ApiError, STORAGE_CLASSES, type Instance } from '@/lib/api'
import PhaseBadge from '@/components/PhaseBadge.vue'
import ExternalBadge from '@/components/ExternalBadge.vue'
import { isValidCidr } from '@/lib/cidr'
import { isValidIdentifier } from '@/lib/identifier'

const instances = ref<Instance[]>([])
const loading = ref(true)
const loadError = ref('')

const showCreateForm = ref(false)
const name = ref('')
const podCount = ref(1)
const storageSize = ref('5Gi')
const storageClass = ref<string>(STORAGE_CLASSES[0])
const database = ref('')
const username = ref('')
const identifierError = ref('')
const allowedIPsInput = ref('')
const allowedIPsError = ref('')
const fetchingIP = ref(false)
const createError = ref('')
const creating = ref(false)

function parseAllowedIPs(): string[] {
  return allowedIPsInput.value
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
}

async function useMyIP() {
  fetchingIP.value = true
  allowedIPsError.value = ''
  try {
    const res = await fetch('https://api.ipify.org?format=json')
    const data = (await res.json()) as { ip: string }
    const cidr = `${data.ip}/32`
    const current = parseAllowedIPs()
    if (!current.includes(cidr)) {
      allowedIPsInput.value = [...current, cidr].join(', ')
    }
  } catch {
    allowedIPsError.value = 'Could not detect your IP — enter it manually.'
  } finally {
    fetchingIP.value = false
  }
}

const deletingId = ref<string | null>(null)

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

async function onCreate() {
  createError.value = ''
  allowedIPsError.value = ''
  identifierError.value = ''

  if (!isValidIdentifier(database.value)) {
    identifierError.value = 'Database name must start with a letter/underscore and contain only letters, digits, underscores.'
    return
  }
  if (!isValidIdentifier(username.value)) {
    identifierError.value = 'Username must start with a letter/underscore and contain only letters, digits, underscores.'
    return
  }

  const allowedIPs = parseAllowedIPs()
  const invalid = allowedIPs.filter((cidr) => !isValidCidr(cidr))
  if (invalid.length > 0) {
    allowedIPsError.value = `Invalid CIDR${invalid.length > 1 ? 's' : ''}: ${invalid.join(', ')}`
    return
  }

  creating.value = true
  try {
    await instancesApi.create({
      name: name.value,
      instances: podCount.value,
      storageSize: storageSize.value,
      storageClass: storageClass.value,
      database: database.value,
      username: username.value,
      allowedIPs: allowedIPs.length > 0 ? allowedIPs : undefined,
    })
    name.value = ''
    podCount.value = 1
    storageSize.value = '5Gi'
    storageClass.value = STORAGE_CLASSES[0]
    database.value = ''
    username.value = ''
    allowedIPsInput.value = ''
    showCreateForm.value = false
    await refresh()
  } catch (e) {
    createError.value = e instanceof ApiError ? e.message : 'Failed to create instance.'
  } finally {
    creating.value = false
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
    <div class="mb-8 flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-foreground">Instances</h1>
      <button
        class="flex items-center gap-1.5 rounded-md bg-accent px-4 py-2.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong"
        @click="showCreateForm = !showCreateForm"
      >
        <svg v-if="!showCreateForm" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round">
          <path d="M12 5v14M5 12h14" />
        </svg>
        {{ showCreateForm ? 'Cancel' : 'New instance' }}
      </button>
    </div>

    <form
      v-if="showCreateForm"
      class="mb-8 max-w-2xl space-y-5 rounded-lg border border-border bg-surface-2 p-7 shadow-sm"
      @submit.prevent="onCreate"
    >
      <div>
        <label for="name" class="mb-1 block text-sm font-medium text-muted">Name</label>
        <input
          id="name"
          v-model="name"
          type="text"
          required
          pattern="^[a-z][a-z0-9-]{0,9}$"
          placeholder="api-test-1"
          class="w-full rounded-md border border-border bg-surface px-3.5 py-2.5 font-mono text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
        />
        <p class="mt-1 text-xs text-faint">
          Lowercase letters, digits, hyphens. Starts with a letter. Max 11 characters.
        </p>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="podCount" class="mb-1 block text-sm font-medium text-muted"
            >Pods</label
          >
          <input
            id="podCount"
            v-model.number="podCount"
            type="number"
            min="1"
            max="5"
            required
            class="w-full rounded-md border border-border bg-surface px-3.5 py-2.5 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>
        <div>
          <label for="storageSize" class="mb-1 block text-sm font-medium text-muted"
            >Storage per pod</label
          >
          <input
            id="storageSize"
            v-model="storageSize"
            type="text"
            required
            placeholder="5Gi"
            class="w-full rounded-md border border-border bg-surface px-3.5 py-2.5 font-mono text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>
      </div>

      <div>
        <label for="storageClass" class="mb-1 block text-sm font-medium text-muted"
          >Storage class</label
        >
        <select
          id="storageClass"
          v-model="storageClass"
          required
          class="w-full rounded-md border border-border bg-surface px-3.5 py-2.5 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
        >
          <option v-for="sc in STORAGE_CLASSES" :key="sc" :value="sc">{{ sc }}</option>
        </select>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="database" class="mb-1 block text-sm font-medium text-muted"
            >Database name</label
          >
          <input
            id="database"
            v-model="database"
            type="text"
            required
            placeholder="mydb"
            class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm font-mono text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>
        <div>
          <label for="username" class="mb-1 block text-sm font-medium text-muted"
            >Username</label
          >
          <input
            id="username"
            v-model="username"
            type="text"
            required
            placeholder="myuser"
            class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm font-mono text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>
      </div>
      <p v-if="identifierError" class="text-xs text-danger">{{ identifierError }}</p>

      <div>
        <div class="mb-1 flex items-center justify-between">
          <label for="allowedIPs" class="block text-sm font-medium text-muted"
            >Allowed IPs <span class="text-faint">(optional)</span></label
          >
          <button
            type="button"
            :disabled="fetchingIP"
            class="flex items-center gap-1 text-xs font-semibold text-accent hover:text-accent-strong disabled:opacity-50"
            @click="useMyIP"
          >
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
              <circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
            </svg>
            {{ fetchingIP ? 'Detecting…' : 'Use my current IP' }}
          </button>
        </div>
        <input
          id="allowedIPs"
          v-model="allowedIPsInput"
          type="text"
          placeholder="203.0.113.0/24, 198.51.100.42/32"
          class="w-full rounded-md border border-border bg-surface px-3.5 py-2.5 font-mono text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
        />
        <p class="mt-1 text-xs text-faint">
          Comma-separated CIDR ranges allowed to reach this instance from outside the cluster.
          Leave empty to keep it internal-only.
        </p>
        <p v-if="allowedIPsError" class="mt-1 text-xs text-danger">{{ allowedIPsError }}</p>
      </div>

      <p v-if="createError" class="text-sm text-danger">{{ createError }}</p>

      <button
        type="submit"
        :disabled="creating"
        class="rounded-md bg-accent px-4 py-2.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
      >
        {{ creating ? 'Creating…' : 'Create' }}
      </button>
    </form>

    <p v-if="loadError" class="mb-4 text-sm text-danger">{{ loadError }}</p>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>

    <div v-else-if="sortedInstances.length === 0" class="rounded-lg border border-dashed border-border-strong px-6 py-12 text-center text-sm text-muted">
      No instances yet. Create one to get started.
    </div>

    <template v-else>
      <div class="mb-4 flex items-center justify-end gap-2.5">
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
          class="flex flex-col gap-4 rounded-lg border border-border bg-surface-2 p-5 shadow-sm transition-colors hover:border-border-strong"
        >
          <div class="flex items-start justify-between gap-2">
            <RouterLink
              :to="`/instances/${instance.id}`"
              class="min-w-0 truncate font-mono text-base font-semibold text-foreground hover:text-accent"
            >
              {{ instance.name }}
            </RouterLink>
            <button
              :disabled="deletingId === instance.id"
              class="shrink-0 rounded-md p-2 text-faint hover:bg-danger-soft hover:text-danger disabled:opacity-50"
              :title="deletingId === instance.id ? 'Deleting…' : 'Delete instance'"
              @click="onDelete(instance.id)"
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
