<template>
  <!-- Loading / error / empty / no-match states of the movie and TV lists; the grid goes in the default slot -->
  <empty-state
    v-if="!total && (refreshCacheLoading || !libraryLoaded)"
    loading
    :title="refreshCacheLoading ? '正在更新缓存…' : '正在加载…'"
    :hint="refreshCacheLoading ? '视频较多时需要一些时间' : ''"
  />
  <empty-state v-else-if="!total && libraryError" icon="error_outline" title="视频列表加载失败" :hint="libraryError">
    <q-btn color="primary" outline label="重试" @click="getLibraryList" />
  </empty-state>
  <empty-state v-else-if="!total" icon="video_library" title="当前没有可用视频" hint="点击“更新缓存”按钮可重建缓存" />
  <empty-state v-else-if="!shown" icon="search_off" :title="`没有匹配“${search}”的视频`" />
  <slot v-else />
</template>

<script setup>
import EmptyState from 'components/EmptyState';
import { getLibraryList, libraryError, libraryLoaded, refreshCacheLoading } from 'pages/library/use-library';

defineProps({
  total: Number,
  shown: Number,
  search: String,
});
</script>
