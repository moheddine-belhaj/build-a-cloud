<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { instancesApi, ApiError, STORAGE_CLASSES } from '@/lib/api'
import DatabaseIcon from '@/components/DatabaseIcon.vue'
import { isValidCidr } from '@/lib/cidr'
import { isValidIdentifier } from '@/lib/identifier'

const router = useRouter()

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
    const created = await instancesApi.create({
      name: name.value,
      instances: podCount.value,
      storageSize: storageSize.value,
      storageClass: storageClass.value,
      database: database.value,
      username: username.value,
      allowedIPs: allowedIPs.length > 0 ? allowedIPs : undefined,
    })
    await router.push(`/instances/${created.id}`)
  } catch (e) {
    createError.value = e instanceof ApiError ? e.message : 'Failed to create instance.'
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-2xl px-6 py-10 lg:px-12">
    <RouterLink to="/instances" class="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground">
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l-7 7 7 7" /></svg>
      Back to instances
    </RouterLink>

    <div class="mb-8 flex items-center gap-2.5">
      <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
        <DatabaseIcon :size="18" />
      </span>
      <h1 class="text-2xl font-semibold text-foreground">New instance</h1>
    </div>

    <form
      class="space-y-5 rounded-lg border border-border bg-surface-2 p-7 shadow-sm"
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

      <div class="flex items-center gap-2">
        <button
          type="submit"
          :disabled="creating"
          class="rounded-md bg-accent px-4 py-2.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
        >
          {{ creating ? 'Creating…' : 'Create instance' }}
        </button>
        <RouterLink
          to="/instances"
          class="rounded-md border border-border px-4 py-2.5 text-sm font-medium text-foreground hover:bg-surface-hover"
        >
          Cancel
        </RouterLink>
      </div>
    </form>
  </div>
</template>
