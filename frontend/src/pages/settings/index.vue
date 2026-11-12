<template>
  <q-page class="settings-page q-pa-md">
    <q-banner v-if="isJobRunning" :inline-actions="$q.screen.gt.xs" rounded class="text-white bg-red q-mb-md">
      <template v-slot:avatar>
        <q-icon name="lock" />
      </template>
      任务进程运行中，配置只能查看，不能修改。停止任务后即可修改并保存。
      <template v-slot:action>
        <q-btn color="white" label="停止任务" outline :loading="stopping" @click="stopJobs" />
        <q-btn color="white" label="去总览页面" flat @click="$router.push('/overview')" />
      </template>
    </q-banner>
    <q-card v-if="isSettingsLoaded" flat>
      <q-tabs
        v-model="tab"
        dense
        no-caps
        mobile-arrows
        active-color="primary"
        indicator-color="primary"
        align="justify"
        narrow-indicator
        style="max-width: 720px"
      >
        <q-tab v-for="t in tabs" :key="t.name" :name="t.name" :label="t.label" />
      </q-tabs>

      <q-separator />

      <q-form greedy @submit="submitAll" @validation-error="handleValidationError">
        <!-- keep-alive: visited tabs stay registered with the form, so saving validates them too -->
        <q-tab-panels
          :key="formVersion"
          v-model="tab"
          animated
          keep-alive
          :class="{ disabled: isJobRunning }"
          :style="{ pointerEvents: isJobRunning ? 'none' : 'default' }"
        >
          <q-tab-panel v-for="t in tabs" :key="t.name" :name="t.name" :data-tab="t.name">
            <component :is="t.component" />
          </q-tab-panel>
        </q-tab-panels>

        <form-submit-area />
      </q-form>
    </q-card>
  </q-page>
</template>

<script setup>
import { computed, ref } from 'vue';
import { useQuasar } from 'quasar';
import BasicSettings from 'pages/settings/SettingsPanelBasic';
import AdvancedSettings from 'pages/settings/SettingsPanelAdvanced';
import EmbySettings from 'pages/settings/SettingsPanelEmby';
import { formModel, formVersion, submitAll, useSettings } from 'pages/settings/use-settings';
import { getJobsStatus, isJobRunning } from 'src/store/systemState';
import ExperimentSettings from 'pages/settings/SettingsPanelExperiment';
import FormSubmitArea from 'pages/settings/FormSubmitArea';
import SubSourceSettings from 'pages/settings/SettingsPanelSubSource';
import NotifySettings from 'pages/settings/SettingsPanelNotify';
import JobApi from 'src/api/JobApi';
import { SystemMessage } from 'src/utils/message';

const $q = useQuasar();

const tabs = [
  { name: 'basic', label: '基础配置', component: BasicSettings },
  { name: 'advanced', label: '进阶配置', component: AdvancedSettings },
  { name: 'subSource', label: '字幕源设置', component: SubSourceSettings },
  { name: 'emby', label: '媒体库设置', component: EmbySettings },
  { name: 'notify', label: '通知', component: NotifySettings },
  { name: 'experiment', label: '实验室', component: ExperimentSettings },
];

const tab = ref('basic');

const isSettingsLoaded = computed(() => Object.keys(formModel).length);

// The failing field may sit in a tab that is not showing; jump to it so the error is visible.
const handleValidationError = (field) => {
  const failedTab = field?.$el?.closest?.('[data-tab]')?.dataset.tab;
  if (failedTab) tab.value = failedTab;
  SystemMessage.error('有配置项填写不正确，请检查标红的输入框', { timeout: 3000 });
};

// Same flow as the 总览 page, so the user does not have to leave to unlock the form.
const stopping = ref(false);
const stopJobs = () => {
  $q.dialog({
    title: '是否强制停止？',
    message: '正在进行的字幕下载任务会被中断。',
    cancel: true,
  }).onOk(async () => {
    stopping.value = true;
    const [, err] = await JobApi.stop();
    stopping.value = false;
    if (err !== null) {
      SystemMessage.error(err.message);
      return;
    }
    getJobsStatus();
    SystemMessage.success('停止成功');
  });
};

useSettings();
</script>

<style lang="scss">
// Not scoped on purpose: shared by every settings panel, namespaced under .settings-page.
.settings-page {
  .settings-list {
    max-width: 720px;
    // dense rows put neighbouring inputs edge to edge
    > .q-item {
      padding-top: 6px;
      padding-bottom: 6px;
    }
  }
  .settings-field {
    width: 220px;
  }
  .settings-field--wide {
    width: 340px;
  }
  .settings-submit {
    position: sticky;
    bottom: 0;
    z-index: 1;
    // theme surface, not #fff: in dark mode the bar stayed white and the light button text vanished
    background: var(--csf-surface);
    border-top: 1px solid rgba(0, 0, 0, 0.12);
  }

  // Phone width: put the control under its label instead of squeezing both into one row.
  @media (max-width: 599px) {
    .q-tab-panel {
      padding: 8px 0;
    }
    .stack-xs {
      flex-wrap: wrap;
      > .q-item__section {
        flex: 1 0 100%;
        min-width: 0;
        align-items: stretch;
      }
      > .q-item__section--side,
      > .q-item__section--avatar {
        padding: 8px 0 0;
      }
    }
    .settings-field,
    .settings-field--wide {
      width: 100%;
    }
  }
}

body.body--dark .settings-page .settings-submit {
  border-top-color: rgba(255, 255, 255, 0.12);
}
</style>
