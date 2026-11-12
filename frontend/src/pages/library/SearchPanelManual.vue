<template>
  <div class="relative-position" style="min-height: 240px">
    <empty-state v-if="error" icon="error_outline" title="获取搜索关键词失败" :hint="error">
      <q-btn color="primary" outline label="重试" @click="getSearchInfo" />
    </empty-state>
    <empty-state
      v-else-if="searchInfo && !searchInfo.search_url?.length"
      icon="search_off"
      title="没有可用的搜索网站"
    />
    <q-list v-else separator>
      <q-item v-for="url in searchInfo?.search_url" :key="url" class="q-px-none">
        <q-item-section>
          <q-item-label class="text-bold q-mb-sm">{{ getDomain(url) }}</q-item-label>
          <div class="row q-gutter-sm">
            <a
              v-for="item in searchInfo?.key_words"
              :key="item"
              :href="getSearchUrl(url, item)"
              target="_blank"
              style="text-decoration: none"
            >
              <q-badge class="cursor-pointer q-pa-xs" outline color="primary" title="点击跳转到网站搜索">
                {{ item }}
                <q-icon name="open_in_new" class="q-ml-xs" />
              </q-badge>
            </a>
          </div>
        </q-item-section>
      </q-item>
    </q-list>
    <q-inner-loading :showing="loading">
      <q-spinner size="40px" color="primary" />
    </q-inner-loading>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue';
import LibraryApi from 'src/api/LibraryApi';
import EmptyState from 'components/EmptyState';

const props = defineProps({
  path: String,
  isMovie: {
    type: Boolean,
    default: false,
  },
});

const searchInfo = ref(null);
const loading = ref(false);
const error = ref('');

const getSearchInfo = async () => {
  loading.value = true;
  const [data, err] = await LibraryApi.getSearchSubtitleInfo({
    video_f_path: props.path,
    is_movie: props.isMovie,
  });
  loading.value = false;
  // shown in place with a retry button instead of a toast over an endless spinner
  error.value = err !== null ? err.message : '';
  searchInfo.value = data;
};

const getDomain = (url) => /https?:\/\/([^/]+)/.exec(url)?.[1] ?? url;

const getSearchUrl = (url, keyword) => {
  if (url.includes('%s')) {
    return url.replace('%s', encodeURIComponent(keyword));
  }
  return url + encodeURIComponent(keyword);
};

onMounted(() => {
  getSearchInfo();
});
</script>
