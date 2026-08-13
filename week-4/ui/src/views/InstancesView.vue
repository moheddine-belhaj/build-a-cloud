<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { instancesApi, ApiError, type Instance } from '@/lib/api'
import PhaseBadge from '@/components/PhaseBadge.vue'
import ExternalBadge from '@/components/ExternalBadge.vue'
import { isValidCidr } from '@/lib/cidr'

const instances = ref<Instance[]>([])
const loading = ref(true)
const loadError = ref('')

const showCreateForm = ref(false)
const name = ref('')
const podCount = ref(1)
const storageSize = ref('5Gi')
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
      allowedIPs: allowedIPs.length > 0 ? allowedIPs : undefined,
    })
    name.value = ''
    podCount.value = 1
    storageSize.value = '5Gi'
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
  <div class="mx-auto max-w-4xl px-4 py-8">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-xl font-semibold text-slate-900">Instances</h1>
      <button
        class="rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700"
        @click="showCreateForm = !showCreateForm"
      >
        {{ showCreateForm ? 'Cancel' : 'New instance' }}
      </button>
    </div>

    <form
      v-if="showCreateForm"
      class="mb-6 space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
      @submit.prevent="onCreate"
    >
      <div>
        <label for="name" class="mb-1 block text-sm font-medium text-slate-700">Name</label>
        <input
          id="name"
          v-model="name"
          type="text"
          required
          pattern="^[a-z][a-z0-9-]{0,9}$"
          placeholder="api-test-1"
          class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
        />
        <p class="mt-1 text-xs text-slate-400">
          Lowercase letters, digits, hyphens. Starts with a letter. Max 11 characters.
        </p>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="podCount" class="mb-1 block text-sm font-medium text-slate-700"
            >Pods</label
          >
          <input
            id="podCount"
            v-model.number="podCount"
            type="number"
            min="1"
            max="5"
            required
            class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
          />
        </div>
        <div>
          <label for="storageSize" class="mb-1 block text-sm font-medium text-slate-700"
            >Storage per pod</label
          >
          <input
            id="storageSize"
            v-model="storageSize"
            type="text"
            required
            placeholder="5Gi"
            class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
          />
        </div>
      </div>

      <div>
        <div class="mb-1 flex items-center justify-between">
          <label for="allowedIPs" class="block text-sm font-medium text-slate-700"
            >Allowed IPs (optional)</label
          >
          <button
            type="button"
            :disabled="fetchingIP"
            class="text-xs font-medium text-slate-500 hover:text-slate-900 disabled:opacity-50"
            @click="useMyIP"
          >
            {{ fetchingIP ? 'Detecting…' : 'Use my current IP' }}
          </button>
        </div>
        <input
          id="allowedIPs"
          v-model="allowedIPsInput"
          type="text"
          placeholder="203.0.113.0/24, 198.51.100.42/32"
          class="w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-slate-500 focus:outline-none"
        />
        <p class="mt-1 text-xs text-slate-400">
          Comma-separated CIDR ranges allowed to reach this instance from outside the cluster.
          Leave empty to keep it internal-only.
        </p>
        <p v-if="allowedIPsError" class="mt-1 text-xs text-red-600">{{ allowedIPsError }}</p>
      </div>

      <p v-if="createError" class="text-sm text-red-600">{{ createError }}</p>

      <button
        type="submit"
        :disabled="creating"
        class="rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"
      >
        {{ creating ? 'Creating…' : 'Create' }}
      </button>
    </form>

    <p v-if="loadError" class="mb-4 text-sm text-red-600">{{ loadError }}</p>

    <div v-if="loading" class="text-sm text-slate-500">Loading…</div>

    <div v-else-if="instances.length === 0" class="text-sm text-slate-500">
      No instances yet. Create one to get started.
    </div>

    <div v-else class="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
      <table class="w-full text-left text-sm">
        <thead class="border-b border-slate-200 bg-slate-50 text-slate-500">
          <tr>
            <th class="px-4 py-2 font-medium">Name</th>
            <th class="px-4 py-2 font-medium">Phase</th>
            <th class="px-4 py-2 font-medium">Access</th>
            <th class="px-4 py-2 font-medium">Pods</th>
            <th class="px-4 py-2 font-medium">Created</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-for="instance in instances" :key="instance.id">
            <td class="px-4 py-2">
              <RouterLink
                :to="`/instances/${instance.id}`"
                class="font-medium text-slate-900 hover:underline"
              >
                {{ instance.name }}
              </RouterLink>
            </td>
            <td class="px-4 py-2"><PhaseBadge :phase="instance.phase" /></td>
            <td class="px-4 py-2"><ExternalBadge :external="instance.external" /></td>
            <td class="px-4 py-2 text-slate-600">
              {{ instance.readyInstances }} / {{ instance.instances }}
            </td>
            <td class="px-4 py-2 text-slate-600">
              {{ new Date(instance.createdAt).toLocaleString() }}
            </td>
            <td class="px-4 py-2 text-right">
              <button
                :disabled="deletingId === instance.id"
                class="text-sm text-red-600 hover:underline disabled:opacity-50"
                @click="onDelete(instance.id)"
              >
                {{ deletingId === instance.id ? 'Deleting…' : 'Delete' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
