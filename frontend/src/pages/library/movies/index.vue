<template>
  <q-page class="q-pa-md">
    <div class="row items-center q-gutter-sm">
      <btn-dialog-library-refresh />
      <btn-dialog-media-server-subtitle-refresh />

      <q-space class="gt-xs" />

      <q-input v-model="filterForm.search" class="col-grow col-sm-auto" outlined dense label="输入关键字搜索">
        <template #append>
          <q-icon name="search" />
        </template>
      </q-input>
    </div>

    <library-list-state :total="movies.length" :shown="filteredMovies.length" :search="filterForm.search">
      <div class="media-grid q-mt-md">
        <q-intersection v-for="item in filteredMovies" once :key="item.video_f_path" style="height: 296px">
          <list-item-movie :data="item" />
        </q-intersection>
      </div>
    </library-list-state>
  </q-page>
</template>

<script setup>
import { useLibrary } from 'pages/library/use-library';
import { computed, reactive } from 'vue';
import BtnDialogLibraryRefresh from 'pages/library/BtnLibraryRefresh';
import BtnDialogMediaServerSubtitleRefresh from 'pages/library/BtnMediaServerSubtitleRefresh';
import LibraryListState from 'pages/library/LibraryListState';
import ListItemMovie from './ListItemMovie';

const filterForm = reactive({
  hasSubtitle: null,
  search: '',
});

const { movies } = useLibrary();

const filteredMovies = computed(() => {
  let res = movies.value;

  if (filterForm.hasSubtitle === true) {
    res = movies.value.filter((item) => item.sub_f_path_list.length > 0);
  }
  if (filterForm.hasSubtitle === false) {
    res = movies.value.filter((item) => item.sub_f_path_list.length === 0);
  }

  if (filterForm.search !== '') {
    res = res.filter((item) => item.name.toLowerCase().includes(filterForm.search.toLowerCase()));
  }

  return res;
});
</script>
