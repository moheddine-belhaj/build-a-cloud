<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { instancesApi, ApiError, type Instance, type ConnectionInfo } from '@/lib/api'
import PhaseBadge from '@/components/PhaseBadge.vue'
import ExternalBadge from '@/components/ExternalBadge.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()

const instance = ref<Instance | null>(null)
const loadError = ref('')
const deleting = ref(false)

const connection = ref<ConnectionInfo | null>(null)
const connectionError = ref('')
const loadingConnection = ref(false)
const showPassword = ref(false)
const copiedConnectionString = ref(false)

const connectionString = computed(() => {
  if (!connection.value) return ''
  const { username, password, host, port, database } = connection.value
  return `postgresql://${encodeURIComponent(username)}:${encodeURIComponent(password)}@${host}:${port}/${database}`
})

let pollHandle: ReturnType<typeof setInterval> | undefined

async function refresh() {
  try {
    instance.value = await instancesApi.get(props.id)
    loadError.value = ''
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to load instance.'
  }
}

async function loadConnection() {
  connectionError.value = ''
  loadingConnection.value = true
  try {
    connection.value = await instancesApi.connection(props.id)
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      // Two distinct "not ready yet" states: the database itself still
      // bootstrapping vs. (for external instances) the LoadBalancer IP not
      // being assigned yet — the backend's message tells them apart.
      connectionError.value = e.message.includes('external endpoint')
        ? "This instance's external endpoint is still being provisioned — try again shortly."
        : "Instance hasn't finished provisioning yet — try again shortly."
    } else {
      connectionError.value = e instanceof ApiError ? e.message : 'Failed to load connection details.'
    }
  } finally {
    loadingConnection.value = false
  }
}

async function copyConnectionString() {
  await copy(connectionString.value)
  copiedConnectionString.value = true
  setTimeout(() => (copiedConnectionString.value = false), 2000)
}

async function onDelete() {
  if (!confirm(`Delete instance "${props.id}"? This cannot be undone.`)) return
  deleting.value = true
  try {
    await instancesApi.remove(props.id)
    await router.push('/instances')
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to delete instance.'
    deleting.value = false
  }
}

function copy(text: string) {
  return navigator.clipboard.writeText(text)
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
  <div class="mx-auto max-w-3xl px-4 py-8">
    <RouterLink to="/instances" class="mb-4 inline-block text-sm text-slate-500 hover:underline"
      >&larr; Back to instances</RouterLink
    >

    <p v-if="loadError" class="mb-4 text-sm text-red-600">{{ loadError }}</p>

    <div v-if="instance" class="space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-xl font-semibold text-slate-900">{{ instance.name }}</h1>
        <div class="flex items-center gap-2">
          <ExternalBadge :external="instance.external" />
          <PhaseBadge :phase="instance.phase" />
        </div>
      </div>

      <div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
        <h2 class="mb-4 text-sm font-semibold text-slate-900">General information</h2>
        <dl class="grid grid-cols-2 gap-4 text-sm">
          <div>
            <dt class="text-slate-500">Instance name</dt>
            <dd class="text-slate-900">{{ instance.name }}</dd>
          </div>
          <div>
            <dt class="text-slate-500">Instance ID</dt>
            <dd class="font-mono text-xs text-slate-900">{{ instance.uid || instance.id }}</dd>
          </div>
          <div>
            <dt class="text-slate-500">Version</dt>
            <dd class="text-slate-900">{{ instance.version || '—' }}</dd>
          </div>
          <div>
            <dt class="text-slate-500">Health</dt>
            <dd><PhaseBadge :phase="instance.phase" /></dd>
          </div>
          <div>
            <dt class="text-slate-500">Size</dt>
            <dd class="text-slate-900">{{ instance.storageSize || '—' }}</dd>
          </div>
          <div>
            <dt class="text-slate-500">Pods ready</dt>
            <dd class="text-slate-900">{{ instance.readyInstances }} / {{ instance.instances }}</dd>
          </div>
          <div>
            <dt class="text-slate-500">Created</dt>
            <dd class="text-slate-900">{{ new Date(instance.createdAt).toLocaleString() }}</dd>
          </div>
        </dl>
      </div>

      <div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-slate-900">Connection details</h2>
          <button
            v-if="!connection"
            :disabled="loadingConnection"
            class="rounded-md bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"
            @click="loadConnection"
          >
            {{ loadingConnection ? 'Loading…' : 'Reveal' }}
          </button>
        </div>

        <p v-if="!connection" class="mb-4 text-sm text-slate-500">
          {{
            instance.external
              ? 'This instance has a public endpoint, reachable from the IP ranges allowed at creation time.'
              : 'This instance is internal-only — reachable from inside the cluster, not from the public internet.'
          }}
        </p>

        <p v-if="connectionError" class="text-sm text-red-600">{{ connectionError }}</p>

        <div v-if="connection" class="mb-4 flex items-center gap-2 rounded-md bg-slate-50 p-3">
          <code class="flex-1 truncate font-mono text-xs text-slate-700">{{ connectionString }}</code>
          <button
            class="shrink-0 rounded-md bg-slate-900 px-2.5 py-1 text-xs font-medium text-white hover:bg-slate-700"
            @click="copyConnectionString"
          >
            {{ copiedConnectionString ? 'Copied!' : 'Copy connection string' }}
          </button>
        </div>

        <dl v-if="connection" class="space-y-3 text-sm">
          <div class="flex items-center justify-between">
            <dt class="text-slate-500">Host</dt>
            <dd class="flex items-center gap-2 font-mono text-slate-900">
              {{ connection.host }}
              <button class="text-slate-400 hover:text-slate-700" @click="copy(connection.host)">
                copy
              </button>
            </dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-slate-500">Port</dt>
            <dd class="font-mono text-slate-900">{{ connection.port }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-slate-500">Database</dt>
            <dd class="font-mono text-slate-900">{{ connection.database }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-slate-500">Username</dt>
            <dd class="font-mono text-slate-900">{{ connection.username }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-slate-500">Password</dt>
            <dd class="flex items-center gap-2 font-mono text-slate-900">
              {{ showPassword ? connection.password : '••••••••••' }}
              <button class="text-slate-400 hover:text-slate-700" @click="showPassword = !showPassword">
                {{ showPassword ? 'hide' : 'show' }}
              </button>
              <button
                class="text-slate-400 hover:text-slate-700"
                @click="copy(connection.password)"
              >
                copy
              </button>
            </dd>
          </div>
        </dl>
      </div>

      <div class="flex justify-end">
        <button
          :disabled="deleting"
          class="rounded-md border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
          @click="onDelete"
        >
          {{ deleting ? 'Deleting…' : 'Delete instance' }}
        </button>
      </div>
    </div>
  </div>
</template>
