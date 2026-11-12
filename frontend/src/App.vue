<template>
  <router-view />
</template>
<script setup>
import { getJobsStatus, systemState } from 'src/store/systemState';
import useInterval from 'src/composables/use-interval';
import { watch } from 'vue';
import { Dark, LocalStorage } from 'quasar';
import { userState } from 'src/store/userState';
import { useAppStatusLoading } from 'src/composables/use-app-status-loading';

// Dark mode: the header toggle stores true/false; with no stored choice follow the OS.
Dark.set(LocalStorage.getItem('darkMode') ?? 'auto');

const { startLoading } = useAppStatusLoading();

const getSystemJobStatus = () => {
  if (userState.accessToken && systemState.systemInfo?.is_setup) {
    getJobsStatus();
  }
};

useInterval(() => {
  getSystemJobStatus();
}, 8000);

watch(
  () => systemState.systemInfo?.is_setup,
  (val) => {
    getSystemJobStatus();
    if (val) {
      startLoading();
    }
  }
);
</script>
