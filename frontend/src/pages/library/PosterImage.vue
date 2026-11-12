<template>
  <!-- Poster with a placeholder icon when there is none, or when it fails to load
       (the backend returns a bare directory url if no poster file exists) -->
  <div class="poster-placeholder q-mb-sm" style="height: 220px">
    <q-img v-if="url" :src="getUrl(url)" class="fit rounded-borders" no-spinner fit="cover">
      <template #error>
        <div class="absolute-full flex flex-center bg-transparent">
          <q-icon :name="icon" size="48px" color="grey" />
        </div>
      </template>
    </q-img>
    <q-icon v-else :name="icon" size="48px" color="grey" />
  </div>
</template>

<script setup>
import { getUrl } from 'pages/library/use-library';

defineProps({
  url: String,
  icon: { type: String, default: 'movie' },
});
</script>
