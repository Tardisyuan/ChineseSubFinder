<template>
  <!-- Shared empty / loading / error placeholder -->
  <div class="column items-center text-center q-pa-xl text-grey">
    <q-spinner v-if="loading" size="40px" color="primary" />
    <q-icon v-else :name="icon" size="56px" />
    <div class="text-subtitle1 q-mt-md">{{ title }}</div>
    <div v-if="hint" class="text-caption q-mt-xs">{{ hint }}</div>
    <div v-if="$slots.default" class="q-mt-md">
      <slot />
    </div>
  </div>
</template>

<script setup>
defineProps({
  icon: { type: String, default: 'inbox' },
  title: String,
  hint: String,
  loading: Boolean,
});
</script>
