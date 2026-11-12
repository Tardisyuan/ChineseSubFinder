<template>
  <q-layout view="lHh Lpr lFf">
    <q-header bordered :class="$q.dark.isActive ? 'bg-dark text-white' : 'bg-white text-dark'">
      <q-toolbar class="q-gutter-x-xs">
        <q-btn flat dense round icon="menu" aria-label="菜单" @click="leftDrawerOpen = !leftDrawerOpen" />
        <q-toolbar-title class="text-subtitle1 text-weight-medium">{{ $route.meta.title }}</q-toolbar-title>
        <!-- labels collapse to icons on phones so the toolbar never overflows -->
        <q-btn
          flat
          dense
          icon="help_outline"
          :label="$q.screen.gt.xs ? '帮助文档' : undefined"
          title="帮助文档"
          @click="openPage('https://github.com/Tardisyuan/ChineseSubFinder/blob/master/docker/readme.md')"
        />
        <BugReportItem />
        <q-btn
          flat
          dense
          round
          :icon="$q.dark.isActive ? 'light_mode' : 'dark_mode'"
          :title="$q.dark.isActive ? '切换到浅色模式' : '切换到深色模式'"
          @click="toggleDark"
        />
        <q-btn-dropdown
          :label="$q.screen.gt.xs ? userState.username : undefined"
          icon="account_circle"
          title="账户"
          flat
          dense
        >
          <q-list dense style="min-width: 100px">
            <q-item clickable v-close-popup @click="logout">
              <q-item-section>退出登录</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
      </q-toolbar>
    </q-header>

    <q-drawer
      v-model="leftDrawerOpen"
      class="csf-drawer q-pa-sm"
      :breakpoint="720"
      :width="250"
      show-if-above
      bordered
      dark
    >
      <div class="q-pt-md q-pb-lg text-center">
        <img src="icons/logo.png" alt="" style="filter: invert(100%); height: 52px" />
        <div class="text-h6 q-mt-xs">ChineseSubFinder</div>
        <div v-if="buildInfo" class="text-caption text-grey-5" :title="systemState.systemInfo?.version">
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
import { LocalStorage, useQuasar } from 'quasar';
import AccessApi from 'src/api/AccessApi';
import BugReportItem from 'layouts/BugReportItem';
import NoticeDialog from 'components/NoticeDialog';

const router = useRouter();
const $q = useQuasar();

const leftDrawerOpen = ref(false);

// The choice is restored in App.vue; without one the app follows the OS scheme.
const toggleDark = () => {
  $q.dark.toggle();
  LocalStorage.set('darkMode', $q.dark.isActive);
};

// Version comes in as "<yyyymmdd>-<sha> Lite" from the CI build (see .github/workflows/docker.yml).
const buildInfo = computed(() => {
  const raw = systemState.systemInfo?.version?.replace(/\s+lite$/i, '');
  if (!raw) return null;
  const m = raw.match(/^(\d{4})(\d{2})(\d{2})-([0-9a-f]{7,})$/);
  // "unknow" is what the backend reports when no version was baked in at build time
  if (!m) return { raw: raw === 'unknow' ? '本地构建' : raw };
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

<style lang="scss">
.csf-drawer {
  background: var(--csf-drawer-bg);
  // on navy the darker brand green is hard to read; active items use the light one
  --q-primary: #6fcfa8;

  .q-item {
    border-radius: 8px;
    margin-bottom: 2px;
  }

  .q-item--active {
    background: rgba(111, 207, 168, 0.14);
  }
}
</style>
