<template>
  <q-btn color="primary" icon="search" size="sm" flat dense v-bind="$attrs" @click="visible = true" title="字幕搜索" />

  <q-dialog v-model="visible" transition-show="slide-up" transition-hide="slide-down" persistent>
    <q-card style="min-width: 70vw">
      <q-card-section>
        <div class="row justify-between items-center">
          <div class="text-h6 text-grey-8">字幕搜索</div>
          <q-btn icon="close" flat round dense @click="visible = false" />
        </div>
        <div class="text-grey">
          点击关键词会在新窗口打开对应网站的搜索结果。想让程序自动挑选字幕，用列表里的“添加到下载队列”。
        </div>
      </q-card-section>
      <q-separator />
      <q-card-section>
        <search-panel-manual :is-movie="isMovie" :path="path" />
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { ref } from 'vue';
import SearchPanelManual from 'pages/library/SearchPanelManual.vue';

defineProps({
  path: String,
  isMovie: {
    type: Boolean,
    default: false,
  },
  searchPackage: {
    type: Boolean,
    default: false,
  },
  season: {
    type: Number,
  },
  episode: {
    type: Number,
  },
  packageEpisodes: {
    type: Array,
  },
});

const visible = ref(false);
</script>
