<template>
  <q-page class="q-pa-md">
    <q-card v-if="systemState.jobStatus" flat>
      <header class="row items-center q-gutter-lg">
        <div>
          <div>
            守护进程：
            <q-badge v-if="isJobRunning" color="positive">运行中</q-badge>
            <q-badge v-else color="grey">未运行</q-badge>
          </div>
          <div class="text-grey">用于处理定时任务、启动扫描程序</div>
        </div>
        <div>
          <q-btn v-if="isJobRunning" label="强制停止" color="negative" @click="stopJobs" :loading="submitting"></q-btn>
          <q-btn v-else label="立即运行" color="primary" @click="startJobs" :loading="submitting"></q-btn>
        </div>
        <q-space />
        <q-btn flat dense icon="refresh" label="刷新" :loading="loading" @click="loadOverview" />
      </header>
    </q-card>

    <template v-if="overview">
      <div class="row q-col-gutter-md q-mt-sm">
        <div v-for="stat in stats" :key="stat.label" class="col-6 col-md-3">
          <q-card flat bordered class="q-pa-md">
            <div class="text-caption text-grey-7">{{ stat.label }}</div>
            <div class="text-h5">{{ stat.value }}</div>
            <div v-if="stat.hint" class="text-caption text-grey">{{ stat.hint }}</div>
          </q-card>
        </div>
      </div>

      <div class="row q-col-gutter-md q-mt-sm">
        <div class="col-12 col-md-7">
          <q-card flat bordered>
            <q-card-section class="row items-center">
              <div class="text-subtitle1">字幕源</div>
              <q-space />
              <div class="text-caption text-grey">
                {{ overview.suppliers_checked_at ? `检查于 ${overview.suppliers_checked_at}，每小时一次` : '尚未检查' }}
              </div>
            </q-card-section>
            <q-separator />
            <q-markup-table flat dense>
              <thead>
                <tr>
                  <th class="text-left">名称</th>
                  <th class="text-left">状态</th>
                  <th class="text-right">响应</th>
                  <th class="text-right">今日下载</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in overview.suppliers" :key="s.name">
                  <td>{{ s.name }}</td>
                  <td>
                    <q-badge :color="s.valid ? 'positive' : 'negative'">{{ s.valid ? '可用' : '不可用' }}</q-badge>
                  </td>
                  <td class="text-right">{{ s.valid ? `${s.speed} ms` : '-' }}</td>
                  <td class="text-right">
                    {{ s.today_count }}<span v-if="s.daily_limit > 0" class="text-grey"> / {{ s.daily_limit }}</span>
                  </td>
                </tr>
                <tr v-if="!overview.suppliers?.length">
                  <td colspan="4" class="text-grey">启动后的第一次检查完成前没有数据</td>
                </tr>
              </tbody>
            </q-markup-table>
          </q-card>
        </div>

        <div class="col-12 col-md-5">
          <q-card flat bordered>
            <q-card-section class="text-subtitle1">外部服务</q-card-section>
            <q-separator />
            <q-list dense>
              <q-item v-for="svc in services" :key="svc.label">
                <q-item-section>{{ svc.label }}</q-item-section>
                <q-item-section side>
                  <q-badge v-if="!svc.enabled" color="grey">未启用</q-badge>
                  <q-badge v-else :color="svc.ok ? 'positive' : 'negative'">{{
                    svc.ok ? '连接正常' : '连接失败'
                  }}</q-badge>
                </q-item-section>
              </q-item>
            </q-list>
            <q-card-section class="text-caption text-grey q-pt-none">结果缓存 2 分钟</q-card-section>
          </q-card>
        </div>
      </div>
    </template>
  </q-page>
</template>

<script setup>
import { getJobsStatus, isJobRunning, systemState } from 'src/store/systemState';
import { useQuasar } from 'quasar';
import { computed, onMounted, ref } from 'vue';
import JobApi from 'src/api/JobApi';
import { SystemMessage } from 'src/utils/message';

const $q = useQuasar();

const submitting = ref(false);
const loading = ref(false);
const overview = ref(null);

const loadOverview = async () => {
  loading.value = true;
  const [res, err] = await JobApi.getOverview();
  loading.value = false;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  overview.value = res;
};

const stats = computed(() => {
  const o = overview.value;
  if (!o) return [];
  const jobs = o.jobs ?? {};
  return [
    { label: '今日下载字幕', value: o.today_downloads, hint: o.today },
    { label: '今日完成任务', value: o.done_today },
    { label: '队列中', value: (jobs.waiting ?? 0) + (jobs.downloading ?? 0), hint: `失败 ${jobs.failed ?? 0}` },
    {
      label: '下次扫描',
      value: o.next_scan_at ? o.next_scan_at.slice(5, 16) : '-',
      hint: o.next_scan_at ? '' : '守护进程未运行',
    },
  ];
});

const services = computed(() => {
  const o = overview.value;
  if (!o) return [];
  return [
    { label: o.media_server.name, ...o.media_server },
    { label: 'TMDB', ...o.tmdb },
  ];
});

const startJobs = () => {
  $q.dialog({
    title: '是否立即运行？',
    cancel: true,
  }).onOk(async () => {
    submitting.value = true;
    const [, err] = await JobApi.start();
    submitting.value = false;
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }
    getJobsStatus();
    SystemMessage.success('启动成功');
  });
};

const stopJobs = () => {
  $q.dialog({
    title: '是否强制停止？',
    cancel: true,
  }).onOk(async () => {
    submitting.value = true;
    const [, err] = await JobApi.stop();
    submitting.value = false;
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }
    getJobsStatus();
    SystemMessage.success('停止成功');
  });
};

onMounted(() => {
  getJobsStatus();
  loadOverview();
});
</script>
