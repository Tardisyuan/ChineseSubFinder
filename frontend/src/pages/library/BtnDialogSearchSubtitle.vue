<template>
  <q-btn color="primary" icon="search" size="sm" flat dense v-bind="$attrs" @click="visible = true" title="字幕搜索" />

  <q-dialog v-model="visible" transition-show="slide-up" transition-hide="slide-down" persistent>
    <q-card style="width: 900px; max-width: min(900px, 95vw)">
      <q-card-section class="q-pb-none">
        <div class="row justify-between items-center no-wrap">
          <div class="text-h6">字幕搜索</div>
          <q-btn icon="close" flat round dense @click="visible = false" title="关闭" />
        </div>
        <div class="text-grey ellipsis" :title="path">{{ path?.split(/\/|\\/).pop() }}</div>
      </q-card-section>
      <q-tabs v-model="tab" dense no-caps align="left" active-color="primary" indicator-color="primary">
        <q-tab name="auto" icon="travel_explore" label="程序搜索" />
        <q-tab name="manual" icon="open_in_new" label="外部网站" />
      </q-tabs>
      <q-separator />
      <q-tab-panels v-model="tab" style="max-height: 65vh" class="scroll">
        <q-tab-panel name="auto">
          <search-panel-auto :is-movie="isMovie" :path="path" />
        </q-tab-panel>
        <q-tab-panel name="manual">
          <div class="text-grey q-mb-sm">点击关键词会在新窗口打开对应网站的搜索结果，下载后用“上传字幕”保存。</div>
          <search-panel-manual :is-movie="isMovie" :path="path" />
        </q-tab-panel>
      </q-tab-panels>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { ref } from 'vue';
import SearchPanelManual from 'pages/library/SearchPanelManual.vue';
import SearchPanelAuto from 'pages/library/SearchPanelAuto.vue';

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
const tab = ref('auto');
</script>
