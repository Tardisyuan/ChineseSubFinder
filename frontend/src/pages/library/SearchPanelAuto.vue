<template>
  <!-- Runs the subtitle sites in the backend for one video and lets the user pick a result -->
  <div class="relative-position" style="min-height: 240px">
    <empty-state v-if="state === null" loading title="正在加载…" />

    <empty-state
      v-else-if="state.status === 'running' && !isMine"
      loading
      title="正在搜索另一个视频"
      :hint="`${fileName(state.video_f_path)}，同一时间只能进行一个搜索，完成后可在这里开始`"
    />

    <empty-state
      v-else-if="state.status === 'running'"
      loading
      :title="`正在各字幕网站搜索… 已用时 ${elapsed} 秒`"
      hint="通常需要几十秒；如果下载队列正在处理任务，会先等它完成。可以关闭窗口，稍后回来查看结果"
    />

    <empty-state
      v-else-if="isMine && state.status === 'failed'"
      icon="error_outline"
      title="搜索失败"
      :hint="state.error"
    >
      <q-btn color="primary" outline label="重试" @click="start" />
    </empty-state>

    <empty-state
      v-else-if="isMine && state.status === 'done' && !state.candidates.length"
      icon="search_off"
      title="各字幕网站都没有找到这个视频的字幕"
      hint="可以换到“外部网站”标签手动查找，下载后用“上传字幕”保存"
    >
      <q-btn color="primary" outline label="重新搜索" @click="start" />
    </empty-state>

    <template v-else-if="isMine && state.status === 'done'">
      <div class="row items-center q-gutter-x-sm q-mb-sm">
        <div class="text-grey">找到 {{ state.candidates.length }} 个字幕，选择一个保存到视频旁</div>
        <q-space />
        <q-btn flat dense color="primary" icon="refresh" label="重新搜索" :disable="!!saving" @click="start" />
      </div>
      <q-checkbox v-model="lockAfterSave" dense class="q-mb-sm" label="保存后锁定该视频（字幕已满意，不再自动更换）" />
      <q-list separator bordered class="rounded-borders">
        <q-expansion-item
          v-for="item in state.candidates"
          :key="item.id"
          group="sub-candidates"
          expand-icon-toggle
          expand-icon="subject"
          expanded-icon="expand_less"
          :title="item.name"
        >
          <template v-slot:header>
            <q-item-section>
              <q-item-label class="candidate-name">{{ item.name }}</q-item-label>
              <q-item-label class="row q-gutter-xs q-mt-xs">
                <q-badge color="primary" outline>{{ item.site || '未知来源' }}</q-badge>
                <q-badge :color="item.is_chinese ? 'teal-7' : 'grey-7'">{{ item.language }}</q-badge>
                <q-badge color="grey-7" outline>{{ item.ext.replace('.', '').toUpperCase() }}</q-badge>
                <q-badge v-if="item.best" color="positive" title="自动下载时程序会选择这一个">
                  <q-icon name="thumb_up" size="12px" class="q-mr-xs" />程序推荐
                </q-badge>
                <q-badge v-else-if="!item.is_chinese" color="orange-9" title="不含中文，自动下载时不会选择">
                  非中文
                </q-badge>
              </q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-btn
                color="primary"
                :outline="!item.best"
                unelevated
                dense
                padding="xs md"
                label="使用"
                title="保存到视频旁"
                :loading="saving === item.id"
                :disable="saving !== null"
                @click.stop="pick(item)"
              />
            </q-item-section>
          </template>
          <pre class="candidate-preview q-ma-none q-pa-md">{{ item.preview.join('\n') || '（没有可预览的内容）' }}</pre>
        </q-expansion-item>
      </q-list>
    </template>

    <empty-state
      v-else
      icon="travel_explore"
      title="让程序去各字幕网站搜索这个视频"
      hint="与自动下载使用相同的字幕源、缓存和每日下载次数；结果列出后由你挑选保存"
    >
      <q-btn color="primary" unelevated icon="search" label="开始搜索" @click="start" />
    </empty-state>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import eventBus from 'vue3-eventbus';
import LibraryApi from 'src/api/LibraryApi';
import EmptyState from 'components/EmptyState';
import { SystemMessage } from 'src/utils/message';
import { VIDEO_TYPE_MOVIE, VIDEO_TYPE_TV } from 'src/constants/SettingConstants';

const props = defineProps({
  path: String,
  isMovie: {
    type: Boolean,
    default: false,
  },
});

// {status: idle|running|done|failed, video_f_path, error, candidates}
const state = ref(null);
const lockAfterSave = ref(true);
const saving = ref(null);
const elapsed = ref(0);
let timer = null;

// the backend keeps one search; it only concerns this dialog when it is for this video
const isMine = computed(() => state.value?.video_f_path === props.path);
const fileName = (path) => path?.split(/\/|\\/).pop();

const stopPolling = () => {
  clearInterval(timer);
  timer = null;
};

const getStatus = async () => {
  const [res, err] = await LibraryApi.getSubtitleSearchStatus();
  if (err !== null) {
    stopPolling();
    state.value = { status: 'failed', video_f_path: props.path, error: err.message, candidates: [] };
    return;
  }
  state.value = res;
  if (res.status === 'running') {
    elapsed.value += 2;
    if (timer === null) {
      timer = setInterval(getStatus, 2000);
    }
  } else {
    stopPolling();
  }
};

const start = async () => {
  elapsed.value = 0;
  const [, err] = await LibraryApi.startSubtitleSearch({
    video_f_path: props.path,
    video_type: props.isMovie ? VIDEO_TYPE_MOVIE : VIDEO_TYPE_TV,
  });
  if (err !== null) {
    state.value = { status: 'failed', video_f_path: props.path, error: err.message, candidates: [] };
    return;
  }
  // "busy" (another video is being searched) shows up through the status as well
  await getStatus();
};

const pick = async (item) => {
  saving.value = item.id;
  const [, err] = await LibraryApi.pickSubtitleSearchResult({
    video_f_path: props.path,
    id: item.id,
    lock: lockAfterSave.value,
  });
  saving.value = null;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  SystemMessage.success(lockAfterSave.value ? '字幕已保存，视频已锁定' : '字幕已保存', { timeout: 2500 });
  eventBus.emit('subtitle-saved', props.path);
  eventBus.emit(`refresh-skip-status-${props.path}`);
};

onMounted(getStatus);
onBeforeUnmount(stopPolling);
</script>

<style lang="scss" scoped>
.candidate-name {
  word-break: break-all;
}

.candidate-preview {
  max-height: 220px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--csf-page-bg);
}
</style>
