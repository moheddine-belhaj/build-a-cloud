<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { instancesApi, ApiError, type Instance, type ConnectionInfo, type ServiceInfo, type UpdateInstanceRequest } from '@/lib/api'
import PhaseBadge from '@/components/PhaseBadge.vue'
import ExternalBadge from '@/components/ExternalBadge.vue'
import CopyButton from '@/components/CopyButton.vue'
import { isValidCidr } from '@/lib/cidr'

const props = defineProps<{ id: string }>()
const router = useRouter()

const instance = ref<Instance | null>(null)
const loadError = ref('')
const deleting = ref(false)

const editing = ref(false)
const editPodCount = ref(1)
const editStorageSize = ref('')
const editAllowedIPsInput = ref('')
const editError = ref('')
const saving = ref(false)

const connection = ref<ConnectionInfo | null>(null)
const connectionError = ref('')
const loadingConnection = ref(false)
const showPassword = ref(false)
const copiedConnectionString = ref(false)

const services = ref<ServiceInfo[]>([])
const servicesError = ref('')
const loadingServices = ref(false)
const servicesLoaded = ref(false)

function portsSummary(svc: ServiceInfo): string {
  if (!svc.ports || svc.ports.length === 0) return '—'
  return svc.ports.map((p) => `${p.port}/${p.protocol ?? 'TCP'}`).join(', ')
}

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

async function loadServices() {
  servicesError.value = ''
  loadingServices.value = true
  try {
    services.value = await instancesApi.services(props.id)
    servicesLoaded.value = true
  } catch (e) {
    servicesError.value = e instanceof ApiError ? e.message : 'Failed to load services.'
  } finally {
    loadingServices.value = false
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

function parseEditAllowedIPs(): string[] {
  return editAllowedIPsInput.value
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
}

function sameIPs(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  const sortedA = [...a].sort()
  const sortedB = [...b].sort()
  return sortedA.every((v, i) => v === sortedB[i])
}

function startEdit() {
  if (!instance.value) return
  editPodCount.value = instance.value.instances
  editStorageSize.value = instance.value.storageSize ?? ''
  editAllowedIPsInput.value = (instance.value.allowedIPs ?? []).join(', ')
  editError.value = ''
  editing.value = true
}

function cancelEdit() {
  editing.value = false
  editError.value = ''
}

async function onSaveEdit() {
  if (!instance.value) return
  editError.value = ''

  const allowedIPs = parseEditAllowedIPs()
  const invalid = allowedIPs.filter((cidr) => !isValidCidr(cidr))
  if (invalid.length > 0) {
    editError.value = `Invalid CIDR${invalid.length > 1 ? 's' : ''}: ${invalid.join(', ')}`
    return
  }

  const body: UpdateInstanceRequest = {}
  if (editPodCount.value !== instance.value.instances) {
    body.instances = editPodCount.value
  }
  if (editStorageSize.value !== (instance.value.storageSize ?? '')) {
    body.storageSize = editStorageSize.value
  }
  if (!sameIPs(allowedIPs, instance.value.allowedIPs ?? [])) {
    body.allowedIPs = allowedIPs
  }

  if (Object.keys(body).length === 0) {
    editError.value = 'No changes to save.'
    return
  }

  saving.value = true
  try {
    instance.value = await instancesApi.update(props.id, body)
    editing.value = false
  } catch (e) {
    editError.value = e instanceof ApiError ? e.message : 'Failed to update instance.'
  } finally {
    saving.value = false
  }
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
  loadServices()
  pollHandle = setInterval(refresh, 5000)
})

onUnmounted(() => {
  clearInterval(pollHandle)
})
</script>

<template>
  <div class="mx-auto max-w-3xl px-6 py-8 lg:px-10">
    <RouterLink to="/instances" class="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground"
      ><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l-7 7 7 7" /></svg>Back to instances</RouterLink
    >

    <p v-if="loadError" class="mb-4 text-sm text-danger">{{ loadError }}</p>

    <div v-if="instance" class="space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="font-mono text-xl font-semibold text-foreground">{{ instance.name }}</h1>
        <div class="flex items-center gap-2">
        <ExternalBadge :external="instance.external" />
          <PhaseBadge :phase="instance.phase" />
        </div>
      </div>

      <div class="rounded-lg border border-border bg-surface-2 p-6 shadow-sm">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">General information</h2>
          <button
            v-if="!editing"
            class="rounded-md bg-accent px-3 py-1.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong"
            @click="startEdit"
          >
            Edit
          </button>
        </div>

        <dl class="grid grid-cols-2 gap-4 text-sm">
          <div>
            <dt class="text-muted">Instance name</dt>
            <dd class="font-mono text-foreground">{{ instance.name }}</dd>
          </div>
          <div>
            <dt class="text-muted">Instance ID</dt>
            <dd class="flex items-center gap-2">
              <span class="truncate font-mono text-xs text-foreground">{{ instance.uid || instance.id }}</span>
              <CopyButton :text="instance.uid || instance.id" />
            </dd>
          </div>
          <div>
            <dt class="text-muted">Version</dt>
            <dd class="text-foreground">{{ instance.version || '—' }}</dd>
          </div>
          <div>
            <dt class="text-muted">Health</dt>
            <dd><PhaseBadge :phase="instance.phase" /></dd>
          </div>
          <div>
            <dt class="text-muted">Size</dt>
            <dd class="font-mono text-foreground">{{ instance.storageSize || '—' }}</dd>
          </div>
          <div>
            <dt class="text-muted">Pods ready</dt>
            <dd class="text-foreground">{{ instance.readyInstances }} / {{ instance.instances }}</dd>
          </div>
          <div>
            <dt class="text-muted">Created</dt>
            <dd class="text-foreground">{{ new Date(instance.createdAt).toLocaleString() }}</dd>
          </div>
        </dl>

        <form v-if="editing" class="mt-6 space-y-4 border-t border-border pt-6" @submit.prevent="onSaveEdit">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="editPodCount" class="mb-1 block text-sm font-medium text-muted">Pods</label>
              <input
                id="editPodCount"
                v-model.number="editPodCount"
                type="number"
                min="1"
                max="5"
                required
                class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
              />
            </div>
            <div>
              <label for="editStorageSize" class="mb-1 block text-sm font-medium text-muted"
                >Storage per pod</label
              >
              <input
                id="editStorageSize"
                v-model="editStorageSize"
                type="text"
                required
                class="w-full rounded-md border border-border bg-surface px-3 py-2 font-mono text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
              />
              <p class="mt-1 text-xs text-faint">
                Can only increase from the current {{ instance.storageSize || 'size' }}, never decrease.
              </p>
            </div>
          </div>

          <div>
            <label for="editAllowedIPs" class="mb-1 block text-sm font-medium text-muted"
              >Allowed IPs</label
            >
            <input
              id="editAllowedIPs"
              v-model="editAllowedIPsInput"
              type="text"
              placeholder="203.0.113.0/24, 198.51.100.42/32"
              class="w-full rounded-md border border-border bg-surface px-3 py-2 font-mono text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
            />
            <p class="mt-1 text-xs text-faint">
              Comma-separated CIDR ranges. Clear this field entirely to remove external access.
            </p>
          </div>

          <p v-if="editError" class="text-sm text-danger">{{ editError }}</p>

          <div class="flex items-center gap-2">
            <button
              type="submit"
              :disabled="saving"
              class="rounded-md bg-accent px-3 py-2 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
            >
              {{ saving ? 'Saving…' : 'Save changes' }}
            </button>
            <button
              type="button"
              :disabled="saving"
              class="rounded-md border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-surface-hover disabled:opacity-50"
              @click="cancelEdit"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>

      <div class="rounded-lg border border-border bg-surface-2 p-6 shadow-sm">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">Connection details</h2>
          <button
            v-if="!connection"
            :disabled="loadingConnection"
            class="rounded-md bg-accent px-3 py-1.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
            @click="loadConnection"
          >
            {{ loadingConnection ? 'Loading…' : 'Reveal' }}
          </button>
        </div>

        <p v-if="!connection" class="mb-4 text-sm text-muted">
          {{
            instance.external
              ? 'This instance has a public endpoint, reachable from the IP ranges allowed at creation time.'
              : 'This instance is internal-only — reachable from inside the cluster, not from the public internet.'
          }}
        </p>

        <p v-if="connectionError" class="text-sm text-danger">{{ connectionError }}</p>

        <div v-if="connection" class="mb-4 flex items-center gap-2 rounded-md border border-border bg-surface p-3">
          <code class="flex-1 truncate font-mono text-xs text-muted">{{ connectionString }}</code>
          <button
            class="flex shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-semibold transition-colors"
            :class="copiedConnectionString ? 'bg-success text-accent-contrast' : 'bg-accent text-accent-contrast hover:bg-accent-strong'"
            @click="copyConnectionString"
          >
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3" /></svg>
            {{ copiedConnectionString ? 'Copied!' : 'Copy connection string' }}
          </button>
        </div>

        <dl v-if="connection" class="space-y-3 text-sm">
          <div class="flex items-start justify-between gap-3">
            <dt class="shrink-0 text-muted">Host</dt>
            <dd class="flex min-w-0 flex-wrap items-center justify-end gap-2 font-mono text-foreground">
              <span class="break-all text-right">{{ connection.host }}</span>
              <CopyButton :text="connection.host" />
            </dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-muted">Port</dt>
            <dd class="font-mono text-foreground">{{ connection.port }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-muted">Database</dt>
            <dd class="font-mono text-foreground">{{ connection.database }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-muted">Username</dt>
            <dd class="font-mono text-foreground">{{ connection.username }}</dd>
          </div>
          <div class="flex items-start justify-between gap-3">
            <dt class="shrink-0 text-muted">Password</dt>
            <dd class="flex min-w-0 flex-wrap items-center justify-end gap-2 font-mono text-foreground">
              <span class="break-all text-right">{{ showPassword ? connection.password : '••••••••••' }}</span>
              <span class="flex shrink-0 items-center gap-2">
                <button class="text-xs text-faint hover:text-foreground" @click="showPassword = !showPassword">
                  {{ showPassword ? 'hide' : 'show' }}
                </button>
                <CopyButton :text="connection.password" />
              </span>
            </dd>
          </div>
        </dl>
      </div>

      <div class="rounded-lg border border-border bg-surface-2 p-6 shadow-sm">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">Network endpoints</h2>
          <button
            v-if="servicesLoaded"
            :disabled="loadingServices"
            class="text-xs font-medium text-muted hover:text-foreground disabled:opacity-50"
            @click="loadServices"
          >
            {{ loadingServices ? 'Refreshing…' : 'Refresh' }}
          </button>
        </div>

        <p class="mb-4 text-sm text-muted">
          Postgres endpoints for this instance: the in-cluster primary (read/write),
          read-only replica, and any-replica Services, plus the external LoadBalancer
          endpoint if this instance has one.
        </p>

        <p v-if="servicesError" class="text-sm text-danger">{{ servicesError }}</p>
        <p v-else-if="loadingServices && !servicesLoaded" class="text-sm text-muted">Loading…</p>
        <p v-else-if="servicesLoaded && services.length === 0" class="text-sm text-muted">No services found.</p>

        <div v-if="services.length > 0" class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead>
              <tr class="border-b border-border text-xs uppercase tracking-wide text-faint">
                <th class="py-2 pr-4 font-medium">Name</th>
                <th class="py-2 pr-4 font-medium">Type</th>
                <th class="py-2 pr-4 font-medium">Cluster IP</th>
                <th class="py-2 pr-4 font-medium">External IP</th>
                <th class="py-2 font-medium">Port(s)</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="svc in services" :key="svc.name" class="border-b border-border last:border-0">
                <td class="py-2.5 pr-4 font-mono text-foreground">{{ svc.name }}</td>
                <td class="py-2.5 pr-4 text-muted">{{ svc.type || '—' }}</td>
                <td class="py-2.5 pr-4 font-mono text-foreground">{{ svc.clusterIP || '—' }}</td>
                <td class="py-2.5 pr-4 font-mono text-muted">{{ svc.externalIP || '—' }}</td>
                <td class="py-2.5 font-mono text-foreground">{{ portsSummary(svc) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="flex justify-end">
        <button
          :disabled="deleting"
          class="rounded-md border border-border px-3 py-2 text-sm font-medium text-danger hover:bg-danger-soft disabled:opacity-50"
          @click="onDelete"
        >
          {{ deleting ? 'Deleting…' : 'Delete instance' }}
        </button>
      </div>
    </div>
  </div>
</template>
