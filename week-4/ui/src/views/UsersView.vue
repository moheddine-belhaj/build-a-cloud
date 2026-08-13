<script setup lang="ts">
// Placeholder view — no backend for user/team management exists yet.
// Deliberately not wired to any API and shows no fake data or buttons.
</script>

<template>
  <div class="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
    <h1 class="mb-6 text-xl font-semibold text-foreground">Users</h1>

    <div class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-border-strong px-6 py-16 text-center">
      <div class="flex h-11 w-11 items-center justify-center rounded-lg border border-border bg-surface-2 text-faint">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="9" cy="8" r="3.2" />
          <path d="M3.5 19c0-3.3 2.5-5.5 5.5-5.5s5.5 2.2 5.5 5.5" />
          <circle cx="17.5" cy="9" r="2.4" />
          <path d="M15.7 13.6c2.3.3 4 2.1 4.1 4.9" />
        </svg>
      </div>
      <p class="text-sm font-semibold text-foreground">Team management is coming soon</p>
      <p class="max-w-sm text-sm text-muted">
        Inviting teammates and assigning roles isn't available yet. This section will let you
        manage who else can see and manage your instances.
      </p>
    </div>
  </div>
</template>
