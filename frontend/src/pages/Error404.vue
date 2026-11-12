<template>
  <div class="window-height column justify-center">
    <empty-state icon="travel_explore" title="404" hint="页面不存在">
      <q-btn color="primary" unelevated label="返回总览" :to="{ name: 'overview' }" />
    </empty-state>
  </div>
</template>

<script setup>
import EmptyState from 'components/EmptyState';
</script>
