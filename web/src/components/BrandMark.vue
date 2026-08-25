<template>
  <span class="brand-mark" aria-hidden="true">
    <svg viewBox="0 0 40 40" fill="none">
      <path d="M7.5 11.5h10l3 3h12v18h-25v-21Z" stroke="currentColor" stroke-width="2" />
      <path d="M13 21h14M13 26h9" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      <circle cx="28.5" cy="8.5" r="2.5" fill="currentColor" />
      <path d="M28.5 11v6.5M28.5 17.5l-5 3M28.5 17.5l5 3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
    </svg>
  </span>
</template>

<style scoped>
.brand-mark {
  display: inline-grid;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  place-items: center;
  color: var(--accent-ink);
  background: var(--accent);
  border: 1px solid color-mix(in srgb, var(--accent) 72%, var(--text));
  border-radius: 9px 9px 9px 3px;
  box-shadow: 5px 5px 0 var(--brand-shadow);
}
.brand-mark svg { width: 31px; height: 31px; }
</style>
