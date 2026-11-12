<template>
  <q-btn label="任务日志" flat dense @click="show = true" color="primary" />

  <q-dialog v-model="show" @before-show="handleBeforeShow">
    <q-card style="width: 900px; max-width: 900px">
      <q-card-section class="row items-center">
        <div class="text-h6">任务日志</div>
        <q-space />
        <q-btn icon="close" flat round dense v-close-popup title="关闭" />
      </q-card-section>

      <q-separator />

      <q-card-section>
        <empty-state v-if="loading" loading title="正在加载日志…" />
        <empty-state v-else-if="!logLines?.length" icon="article" title="这个任务还没有日志" />
        <log-viewer-raw v-else :log-lines="logLines" style="height: 60vh" />
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup>
import JobApi from 'src/api/JobApi';
import { ref } from 'vue';
import { SystemMessage } from 'src/utils/message';
import LogViewerRaw from 'components/LogViewerRaw';
import EmptyState from 'components/EmptyState';

const props = defineProps({
  data: {
    type: Object,
  },
});

const show = ref(false);
const logLines = ref([]);
const loading = ref(false);

const getJobLog = async () => {
  loading.value = true;
  const [res, err] = await JobApi.getLog(props.data.id);
  loading.value = false;
  if (err != null) {
    SystemMessage.error(err.message);
  } else {
    logLines.value = res?.one_line;
  }
};

const handleBeforeShow = () => {
  getJobLog();
};
</script>
