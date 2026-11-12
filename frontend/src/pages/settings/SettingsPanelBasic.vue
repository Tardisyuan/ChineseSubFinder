<template>
  <div>
    <q-list class="settings-list" dense>
      <q-item-label header>媒体目录</q-item-label>

      <q-item v-for="group in pathGroups" :key="group.field" class="stack-xs q-mb-sm">
        <q-item-section top>
          <q-item-label>{{ group.label }}</q-item-label>
          <q-item-label caption>
            {{ isRunningInDocker ? '填容器内的路径，必须在 /media 目录下' : '程序会扫描这些目录里的视频' }}
          </q-item-label>
        </q-item-section>
        <q-item-section side top>
          <div v-for="(item, i) in form[group.field]" :key="i" class="row no-wrap items-start q-gutter-x-xs">
            <q-input
              v-model="form[group.field][i]"
              class="settings-field--wide col"
              :placeholder="group.placeholder"
              standout
              dense
              lazy-rules
              :rules="[(val) => !!val || '目录不能为空', validateRemotePath]"
            />
            <q-btn
              v-if="form[group.field].length > 1"
              class="q-mt-xs"
              icon="remove_circle_outline"
              color="negative"
              flat
              dense
              round
              title="删除"
              @click="form[group.field].splice(i, 1)"
            ></q-btn>
          </div>
          <q-btn
            class="self-start"
            icon="add"
            label="添加目录"
            color="primary"
            flat
            dense
            @click="form[group.field].push('')"
          ></q-btn>
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>扫描计划</q-item-label>

      <q-item class="stack-xs">
        <q-item-section>
          <q-radio v-model="scanType" :val="0" label="按固定间隔扫描" />
        </q-item-section>
        <q-item-section side>
          <q-select
            v-model="scanCronString0"
            class="settings-field"
            :options="scanIntervalOptions"
            standout
            dense
            hide-bottom-space
            :rules="[(val) => !!val || '请选择扫描间隔']"
            emit-value
            map-options
            :disable="scanType !== 0"
            @update:model-value="handleScanIntervalChange"
          />
        </q-item-section>
      </q-item>

      <q-item class="stack-xs">
        <q-item-section>
          <q-radio v-model="scanType" :val="1" label="每天固定时间扫描" />
        </q-item-section>
        <q-item-section side>
          <q-select
            v-model="scanCronString1"
            class="settings-field"
            :options="scanSpecTimeOptions"
            standout
            dense
            hide-bottom-space
            emit-value
            map-options
            hint="最多选择 4 个时间点"
            :rules="[
              (val) => !!val || !!val?.length || '请选择时间点',
              (val) => val.length <= 4 || '最多选择4个时间点',
            ]"
            :disable="scanType !== 1"
            @update:model-value="handleScanSpecTimeChange"
            multiple
          />
        </q-item-section>
      </q-item>

      <q-item class="stack-xs">
        <q-item-section>
          <q-radio v-model="scanType" :val="2" label="自定义 cron 规则" />
          <q-item-label caption class="q-pl-xl">
            规则写法参考
            <a href="https://pkg.go.dev/github.com/robfig/cron/v3" target="_blank" class="text-primary"
              >robfig/cron 文档</a
            >
          </q-item-label>
        </q-item-section>
        <q-item-section side>
          <q-input
            v-model="scanCronString2"
            class="settings-field"
            placeholder="例如 0 3 * * *"
            standout
            dense
            hide-bottom-space
            :rules="[(val) => !!val || '请填写 cron 规则', validateCronTime]"
            @update:model-value="handleScanCustomChange"
            :disable="scanType !== 2"
          />
        </q-item-section>
      </q-item>

      <q-item>
        <q-item-section>
          <q-radio v-model="scanType" :val="3" label="不自动扫描" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>性能</q-item-label>

      <q-item class="stack-xs">
        <q-item-section>
          <q-item-label>并发线程数</q-item-label>
          <q-item-label caption>同时处理的视频数量，设备性能一般就选少一些</q-item-label>
        </q-item-section>
        <q-item-section side>
          <div class="row">
            <q-radio v-model="form.threads" :val="1" label="弱鸡（1线程）" />
            <q-radio v-model="form.threads" :val="3" label="一般（3线程）" />
            <q-radio v-model="form.threads" :val="6" label="超猛（6线程）" />
          </div>
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { validateCronTime, validateRemotePath } from 'src/utils/quasar-validators';
import { toRefs } from '@vueuse/core';
import { ref, watch } from 'vue';
import { isRunningInDocker } from 'src/store/systemState';

const { common_settings: form } = toRefs(formModel);

const pathGroups = [
  { field: 'movie_paths', label: '电影目录', placeholder: '/media/电影' },
  { field: 'series_paths', label: '连续剧目录', placeholder: '/media/连续剧' },
];

const NO_SCAN_CRON_RULE = '@every 87600h';

const scanCronString0 = ref('');
const scanCronString1 = ref([]);
const scanCronString2 = ref('');
const scanType = ref(0);

if (form.value.scan_interval === NO_SCAN_CRON_RULE) {
  scanType.value = 3;
} else if (form.value.interval_or_assign_or_custom === 0) {
  scanType.value = 0;
  scanCronString0.value = form.value.scan_interval.split(' ').pop();
} else if (form.value.interval_or_assign_or_custom === 1) {
  scanType.value = 1;
  scanCronString1.value = form.value.scan_interval.split(' ')[1].split(',');
} else if (form.value.interval_or_assign_or_custom === 2) {
  scanType.value = 2;
  scanCronString2.value = form.value.scan_interval;
}

const scanIntervalOptions = [
  { label: '每4小时', value: '4h' },
  { label: '每5小时', value: '5h' },
  { label: '每6小时', value: '6h' },
  { label: '每7小时', value: '7h' },
  { label: '每8小时', value: '8h' },
  { label: '每9小时', value: '9h' },
  { label: '每10小时', value: '10h' },
];
const scanSpecTimeOptions = [
  { label: '00:00', value: '0' },
  { label: '01:00', value: '1' },
  { label: '02:00', value: '2' },
  { label: '03:00', value: '3' },
  { label: '04:00', value: '4' },
  { label: '05:00', value: '5' },
  { label: '06:00', value: '6' },
  { label: '07:00', value: '7' },
  { label: '08:00', value: '8' },
  { label: '09:00', value: '9' },
  { label: '10:00', value: '10' },
  { label: '11:00', value: '11' },
  { label: '12:00', value: '12' },
  { label: '13:00', value: '13' },
  { label: '14:00', value: '14' },
  { label: '15:00', value: '15' },
  { label: '16:00', value: '16' },
  { label: '17:00', value: '17' },
  { label: '18:00', value: '18' },
  { label: '19:00', value: '19' },
  { label: '20:00', value: '20' },
  { label: '21:00', value: '21' },
  { label: '22:00', value: '22' },
  { label: '23:00', value: '23' },
];

const handleScanIntervalChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 0;
  formModel.common_settings.scan_interval = `@every ${scanCronString0.value}`;
};

const handleScanSpecTimeChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 1;
  formModel.common_settings.scan_interval = `0 ${scanCronString1.value.join(',')} * * *`;
};

const handleScanCustomChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 2;
  formModel.common_settings.scan_interval = `${scanCronString2.value}`;
};

const handleScanNoScanChange = () => {
  formModel.common_settings.interval_or_assign_or_custom = 2;
  formModel.common_settings.scan_interval = NO_SCAN_CRON_RULE;
};

// Picking a radio must take effect on its own, not only when the field next to it is edited.
// An empty field writes nothing; its required rule then blocks saving.
watch(scanType, (type) => {
  if (type === 0 && scanCronString0.value) handleScanIntervalChange();
  if (type === 1 && scanCronString1.value.length) handleScanSpecTimeChange();
  if (type === 2 && scanCronString2.value) handleScanCustomChange();
  if (type === 3) handleScanNoScanChange();
});

// 同步更新emby的线程设置
watch(
  () => formModel.common_settings.threads,
  (val) => {
    formModel.emby_settings.threads = val;
  }
);
</script>
