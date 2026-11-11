<template>
  <div>
    <q-list dense style="max-width: 600px">
      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>连接媒体服务器</q-item-label>
          <q-item-label caption>从 Emby / Jellyfin 获取最近新增的视频，下载字幕后通知它刷新</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.enable">
        <q-item>
          <q-item-section>
            <q-item-label>服务器类型</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-select
              v-model="form.server_type"
              :options="serverTypeOptions"
              emit-value
              map-options
              standout
              dense
              style="min-width: 180px"
            />
          </q-item-section>
        </q-item>
        <q-item>
          <q-item-section>
            <q-item-label>服务器内网地址</q-item-label>
            <q-item-label caption>例如 http://192.168.1.10:8096</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form.address_url"
              standout
              dense
              :rules="[
                (val) => (form.enable && !!val) || '不能为空',
                (val) => /^https?:\/\/[\w.-]+(:[0-9]+)?(\/[\w.-]*)*\/?$/.test(val) || '请输入正确的URL',
              ]"
            />
          </q-item-section>
        </q-item>
        <q-item>
          <q-item-section>
            <q-item-label>API Key</q-item-label>
            <q-item-label caption>{{ apiKeyHint }}</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input v-model="form.api_key" standout dense :rules="[(val) => (form.enable && !!val) || '不能为空']" />
          </q-item-section>
        </q-item>

        <q-item> <btn-check-emby-server /> </q-item>
        <q-item>
          <q-item-section>
            <q-item-label>获取最多的剧集数量</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model.number="form.max_request_video_number"
              standout
              dense
              :rules="[(val) => (form.enable && !!val) || '不能为空', (val) => /^\d+$/.test(val) || '必须是整数']"
            />
          </q-item-section>
        </q-item>
        <q-item tag="label" v-ripple>
          <q-item-section>
            <q-item-label>是否跳过已观看的</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-toggle v-model="form.skip_watched" />
          </q-item-section>
        </q-item>

        <q-separator spaced inset></q-separator>

        <q-item :class="{ disabled: form.auto_or_manual }" tag="label" v-ripple>
          <q-item-section>
            <q-item-label>自动匹配IMDB ID</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-toggle v-model="form.auto_or_manual" :disable="form.auto_or_manual" />
          </q-item-section>
        </q-item>

        <q-separator spaced inset></q-separator>

        <q-item>
          <q-item-section>
            <q-item-label>路径映射</q-item-label>
            <q-item-label caption>
              下载完字幕后会通知 {{ form.server_type === 'jellyfin' ? 'Jellyfin' : 'Emby' }}
              刷新视频所在目录。如果它看到的路径和这里不同（两个容器挂载点不一样），在右边填它看到的路径；留空表示相同。
            </q-item-label>
          </q-item-section>
        </q-item>
        <q-item v-for="row in mappingRows" :key="`${row.kind}:${row.path}`">
          <q-item-section>
            <q-item-label>{{ row.path }}</q-item-label>
            <q-item-label caption>{{ row.kind === 'movie' ? '电影目录' : '连续剧目录' }}</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-input
              v-model="form[row.field][row.path]"
              :placeholder="row.path"
              standout
              dense
              style="min-width: 240px"
            />
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import { computed, watchEffect } from 'vue';
import BtnCheckEmbyServer from 'pages/settings/BtnCheckEmbyServer';

const { emby_settings: form } = toRefs(formModel);

// One row per movie/series folder from 基础配置; use-settings.js keeps the mapping keys in sync.
const mappingRows = computed(() => [
  ...Object.keys(form.value?.movie_paths_mapping ?? {}).map((path) => ({
    kind: 'movie',
    field: 'movie_paths_mapping',
    path,
  })),
  ...Object.keys(form.value?.series_paths_mapping ?? {}).map((path) => ({
    kind: 'series',
    field: 'series_paths_mapping',
    path,
  })),
]);

const serverTypeOptions = [
  { label: 'Emby', value: 'emby' },
  { label: 'Jellyfin', value: 'jellyfin' },
];

// Configs saved before server_type existed have no value; they are Emby setups.
watchEffect(() => {
  if (form.value && !form.value.server_type) form.value.server_type = 'emby';
});

const apiKeyHint = computed(() =>
  form.value?.server_type === 'jellyfin'
    ? 'Jellyfin 控制台 → API 密钥 → 新建'
    : 'Emby 设置 → 高级 → API 密钥 → 新 API 密钥'
);
</script>
