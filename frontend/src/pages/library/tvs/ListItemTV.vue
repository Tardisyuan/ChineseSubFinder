<template>
  <q-card flat class="bg-transparent">
    <poster-image :url="posterInfo?.url" icon="live_tv" />
    <div class="text-ellipsis-line-2" :title="data.name">{{ data.name }}</div>
    <div class="row items-center">
      <q-space />
      <div>
        <dialog-t-v-detail :data="detailInfo">
          <q-btn
            v-if="hasSubtitleVideoCount > 0"
            color="positive"
            flat
            dense
            icon="closed_caption"
            :label="`${hasSubtitleVideoCount}/${detailInfo.one_video_info.length}`"
            title="已有字幕"
          />
          <q-btn v-else color="grey" round flat dense icon="closed_caption" title="没有字幕" />
        </dialog-t-v-detail>
      </div>
    </div>
  </q-card>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import DialogTVDetail from 'pages/library/tvs/DialogTVDetail';
import LibraryApi from 'src/api/LibraryApi';
import { subtitleUploadList } from 'pages/library/use-library';
import PosterImage from 'pages/library/PosterImage';

const props = defineProps({
  data: Object,
});

const posterInfo = ref(null);
const detailInfo = ref(null);

const getPosterInfo = async () => {
  const [res] = await LibraryApi.getTvPoster({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    root_dir_path: props.data.root_dir_path,
  });
  posterInfo.value = res;
};

const getDetailInfo = async () => {
  const [res] = await LibraryApi.getTvDetail({
    name: props.data.name,
    main_root_dir_f_path: props.data.main_root_dir_f_path,
    root_dir_path: props.data.root_dir_path,
  });
  detailInfo.value = res;
};

const hasSubtitleVideoCount = computed(
  () => detailInfo.value?.one_video_info.filter((e) => e.sub_f_path_list.length > 0).length
);

watch(subtitleUploadList, (val, oldValue) => {
  // 上传字幕列表当前文件有变化时刷新
  if (
    detailInfo.value?.one_video_info.some((e) => oldValue.map((f) => f.video_f_path).includes(e.video_f_path)) &&
    !detailInfo.value?.one_video_info.some((e) => val.map((f) => f.video_f_path).includes(e.video_f_path))
  ) {
    getDetailInfo();
  }
});

onMounted(() => {
  getPosterInfo();
  getDetailInfo();
});
</script>

<style lang="scss" scoped>
.text-ellipsis-line-2 {
  height: 40px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
</style>
