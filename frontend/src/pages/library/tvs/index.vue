<template>
  <q-page class="q-pa-md">
    <div class="row items-center q-gutter-sm">
      <btn-dialog-library-refresh />
      <btn-dialog-media-server-subtitle-refresh />
      <q-btn color="primary" outline icon="lock" @click="setLock(true)" label="锁定" />
      <q-btn color="primary" outline icon="lock_open" @click="setLock(false)" label="取消锁定" />
      <div v-if="selection.length" class="text-grey">已选中{{ selection.length }}项</div>
      <q-btn v-if="selection.length" color="primary" flat @click="selection = []" label="清空选择" />

      <q-space class="gt-xs" />

      <q-input v-model="filterForm.search" class="col-grow col-sm-auto" outlined dense label="输入关键字搜索">
        <template #append>
          <q-icon name="search" />
        </template>
      </q-input>
    </div>

    <library-list-state :total="tvs.length" :shown="filteredTvs.length" :search="filterForm.search">
      <div class="media-grid q-mt-md">
        <q-intersection v-for="item in filteredTvs" :key="item.root_dir_path" style="height: 304px" once>
          <div
            class="item-wrapper cursor-pointer relative-position"
            @click="toggleSelection(item)"
            :class="{ selected: selection.includes(item.root_dir_path) }"
          >
            <list-item-t-v :data="item" :selected="selection" />
            <q-checkbox
              :model-value="selection.includes(item.root_dir_path)"
              class="absolute-top-right no-pointer-events"
            />
          </div>
        </q-intersection>
      </div>
    </library-list-state>
  </q-page>
</template>

<script setup>
import { useLibrary } from 'pages/library/use-library';
import { computed, reactive, ref } from 'vue';
import ListItemTV from 'pages/library/tvs/ListItemTV';
import LibraryListState from 'pages/library/LibraryListState';
import BtnDialogLibraryRefresh from 'pages/library/BtnLibraryRefresh';
import BtnDialogMediaServerSubtitleRefresh from 'pages/library/BtnMediaServerSubtitleRefresh';
import { SystemMessage } from 'src/utils/message';
import LibraryApi from 'src/api/LibraryApi';
import { VIDEO_TYPE_TV } from 'src/constants/SettingConstants';
import { useQuasar } from 'quasar';

const $q = useQuasar();

const filterForm = reactive({
  hasSubtitle: null,
  search: '',
});

const selection = ref([]);

const { tvs } = useLibrary();

const filteredTvs = computed(() => {
  let res = tvs.value;

  const getSubtitleCount = (item) => item.one_video_info.filter((e) => e.sub_f_path_list.length > 0).length;

  if (filterForm.hasSubtitle === 1) {
    res = res.filter((item) => getSubtitleCount(item) === 0);
  }
  if (filterForm.hasSubtitle === 2) {
    res = res.filter((item) => getSubtitleCount(item) > 0 && getSubtitleCount(item) < item.one_video_info.length);
  }
  if (filterForm.hasSubtitle === 3) {
    res = res.filter((item) => getSubtitleCount(item) === item.one_video_info.length);
  }

  if (filterForm.search !== '') {
    res = res.filter((item) => item.name.toLowerCase().includes(filterForm.search.toLowerCase()));
  }

  return res;
});

const toggleSelection = (item) => {
  const itemKey = item.root_dir_path;
  if (selection.value.includes(itemKey)) {
    selection.value = selection.value.filter((e) => e !== itemKey);
  } else {
    selection.value.push(itemKey);
  }
};

const lockTv = async (item, lock) => {
  const [tvInfo] = await LibraryApi.getTvDetail({
    name: item.name,
    main_root_dir_f_path: item.main_root_dir_f_path,
    root_dir_path: item.root_dir_path,
  });

  return LibraryApi.setSkipInfo({
    video_skip_infos: tvInfo.one_video_info.map((e) => ({
      video_type: VIDEO_TYPE_TV,
      physical_video_file_full_path: e.video_f_path,
      is_bluray: false,
      is_skip: lock,
    })),
  });
};

const setLock = async (flag) => {
  if (selection.value.length === 0) {
    SystemMessage.warn('请至少选择一项！');
    return;
  }
  $q.dialog({
    title: '提示',
    message: `确定${flag ? '锁定' : '取消锁定'}选中的${selection.value.length}项吗？`,
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    await Promise.allSettled(
      selection.value.map((e) => tvs.value.find((f) => f.root_dir_path === e)).map((e) => lockTv(e, flag))
    );
    // 取消选中
    selection.value = [];
    SystemMessage.success('操作成功！');
  });
};
</script>

<style lang="scss">
.item-wrapper {
  overflow: hidden;
  border-radius: 6px;
  padding: 4px;

  &.selected {
    box-shadow: 0 0 0 2px $primary;
  }
}
</style>
