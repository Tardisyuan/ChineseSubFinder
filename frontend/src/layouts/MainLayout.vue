<template>
  <q-layout view="lHh Lpr lFf">
    <q-header elevated>
      <q-toolbar class="text-white text-primary">
        <q-btn flat dense round color="white" icon="menu" aria-label="Menu" @click="leftDrawerOpen = !leftDrawerOpen" />
        <div class="text-h6 q-ml-sm">{{ $route.meta.title }}</div>
        <q-space />
        <q-item
          clickable
          @click="openPage('https://github.com/Tardisyuan/ChineseSubFinder/blob/master/docker/readme.md')"
        >
          <q-item-section> 帮助文档 </q-item-section>
        </q-item>
        <BugReportItem />
        <q-btn-dropdown :label="userState.username" icon="account_circle" flat>
          <q-list dense style="min-width: 100px">
            <q-item clickable v-close-popup>
              <q-item-section @click="logout">退出登录</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
      </q-toolbar>
    </q-header>

    <q-drawer
      v-model="leftDrawerOpen"
      class="q-pa-md"
      :breakpoint="720"
      :width="280"
      show-if-above
      bordered
      dark
      style="background: #111729"
      content-class="bg-white"
    >
      <div class="text-h5 q-py-sm q-px-md">
        <div class="text-center">
          <img src="icons/logo.png" alt="" style="filter: invert(100%); height: 60px" />
        </div>
        <div class="q-mt-sm text-center">ChineseSubFinder</div>
        <div v-if="buildInfo" class="text-center text-caption text-grey-5" :title="systemState.systemInfo?.version">
          <a
            v-if="buildInfo.commit"
            class="text-grey-5"
            style="text-decoration: none"
            :href="buildInfo.commitUrl"
            target="_blank"
          >
            私有版 {{ buildInfo.date }} · {{ buildInfo.commit }}
          </a>
          <span v-else>{{ buildInfo.raw }}</span>
        </div>
      </div>
      <q-list>
        <menu-item v-for="route in menus" :menu-info="route" :key="`${route.name}${route.path}`" />
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />

      <notice-dialog />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import routes from 'src/router/routes';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import MenuItem from 'layouts/MenuItem';
import { systemState } from 'src/store/systemState';
import { userState } from 'src/store/userState';
import { LocalStorage } from 'quasar';
import AccessApi from 'src/api/AccessApi';
import BugReportItem from 'layouts/BugReportItem';
import NoticeDialog from 'components/NoticeDialog';

const router = useRouter();

const leftDrawerOpen = ref(false);

// Version comes in as "<yyyymmdd>-<sha> Lite" from the CI build (see .github/workflows/docker.yml).
const buildInfo = computed(() => {
  const raw = systemState.systemInfo?.version?.replace(/\s+lite$/i, '');
  if (!raw) return null;
  const m = raw.match(/^(\d{4})(\d{2})(\d{2})-([0-9a-f]{7,})$/);
  if (!m) return { raw };
  return {
    date: `${m[1]}-${m[2]}-${m[3]}`,
    commit: m[4],
    commitUrl: `https://github.com/Tardisyuan/ChineseSubFinder/commit/${m[4]}`,
  };
});
const menus = routes.find((e) => e.path === '/').children;

const logout = () => {
  userState.username = '';
  userState.accessToken = undefined;
  LocalStorage.remove('token');
  AccessApi.logout();
  router.push('/access/login');
};

const openPage = (url) => {
  window.open(url, '_blank');
};
</script>
