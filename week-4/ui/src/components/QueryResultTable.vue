<script setup lang="ts">
import { computed } from 'vue'
import type { QueryResult } from '@/lib/api'

const props = defineProps<{
  result: QueryResult | null
  error?: string
  loading?: boolean
}>()

function formatCell(value: unknown): string {
  if (value === null || value === undefined) return '∅'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const hasColumns = computed(() => !!props.result?.columns && props.result.columns.length > 0)
</script>

<template>
  <div class="text-sm">
    <p v-if="loading" class="text-muted">Running query…</p>

    <p v-else-if="error" class="rounded-md border border-danger/30 bg-danger-soft px-3 py-2 text-danger">{{ error }}</p>

    <p v-else-if="!result" class="text-muted">Run a query to see results here.</p>

    <div v-else-if="hasColumns">
      <div v-if="!result!.rows || result!.rows.length === 0" class="text-muted">Query returned no rows.</div>
      <div v-else class="max-h-96 overflow-auto rounded-md border border-border">
        <table class="w-full border-collapse text-left font-mono text-xs">
          <thead class="sticky top-0 bg-surface-2">
            <tr class="border-b border-border text-faint uppercase tracking-wide">
              <th v-for="col in result!.columns" :key="col" class="whitespace-nowrap px-3 py-2 font-medium">{{ col }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in result!.rows" :key="i" class="border-b border-border last:border-0">
              <td v-for="(cell, j) in row" :key="j" class="whitespace-nowrap px-3 py-1.5 text-foreground">{{ formatCell(cell) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-2 text-xs text-faint">{{ result!.rows?.length ?? 0 }} row(s)</p>
    </div>

    <p v-else class="rounded-md border border-success/30 bg-success-soft px-3 py-2 text-success">
      {{ result.command || 'Statement executed.' }}
      <span v-if="result.rowsAffected !== undefined">— {{ result.rowsAffected }} row(s) affected</span>
    </p>
  </div>
</template>
