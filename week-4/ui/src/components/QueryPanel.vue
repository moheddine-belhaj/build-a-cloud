<script setup lang="ts">
import { computed, ref } from 'vue'
import { instancesApi, ApiError, type QueryResult } from '@/lib/api'
import QueryResultTable from '@/components/QueryResultTable.vue'

const props = defineProps<{ instanceId: string }>()

const query = ref('')
const running = ref(false)
const result = ref<QueryResult | null>(null)
const error = ref('')

// Server-side enforcement is the real guard: every run here sends
// readOnly: true, so Postgres itself rejects any write inside a
// `BEGIN READ ONLY` transaction (see internal/handlers/query.go). This is
// just a friendlier, immediate client-side heads-up on top of that.
const writeKeywords = ['insert', 'update', 'delete', 'drop', 'alter', 'truncate', 'create', 'grant', 'revoke', 'copy', 'vacuum', 'merge']

const looksLikeWrite = computed(() => {
  const firstWord = query.value.trim().split(/\s+/)[0]?.toLowerCase() ?? ''
  return writeKeywords.includes(firstWord)
})

const canRun = computed(() => query.value.trim().length > 0 && !looksLikeWrite.value && !running.value)

async function runQuery() {
  if (!canRun.value) return
  running.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await instancesApi.runQuery(props.instanceId, { query: query.value, readOnly: true })
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Failed to run query.'
  } finally {
    running.value = false
  }
}
</script>

<template>
  <div>
    <div class="overflow-hidden rounded-lg border border-border-strong bg-surface shadow-sm">
      <textarea
        v-model="query"
        rows="14"
        spellcheck="false"
        placeholder="SELECT * FROM my_table LIMIT 100"
        class="block min-h-[22rem] w-full resize-y bg-transparent px-4 py-4 font-mono text-[13.5px] leading-relaxed text-foreground focus:outline-none"
        @keydown.meta.enter="runQuery"
        @keydown.ctrl.enter="runQuery"
      ></textarea>
      <div class="flex items-center justify-between gap-2 border-t border-border bg-surface-2 px-4 py-3">
        <span class="text-xs" :class="looksLikeWrite ? 'text-warning' : 'text-faint'">
          {{ looksLikeWrite ? 'Only read (SELECT) queries are allowed here — this looks like a write statement.' : 'Write statements (INSERT/UPDATE/DELETE/DDL) are rejected.' }}
        </span>
        <div class="flex shrink-0 items-center gap-3">
          <span class="text-xs text-faint">⌘/Ctrl + Enter</span>
          <button
            type="button"
            :disabled="!canRun"
            class="rounded-md bg-accent px-4 py-1.5 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
            @click="runQuery"
          >
            {{ running ? 'Running…' : 'Run query' }}
          </button>
        </div>
      </div>
    </div>

    <div class="mt-6">
      <h2 class="mb-3 text-sm font-semibold text-foreground">Results</h2>
      <QueryResultTable :result="result" :error="error" :loading="running" />
    </div>
  </div>
</template>
