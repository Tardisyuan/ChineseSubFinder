<template>
  <div class="settings-submit q-pa-md row items-center q-gutter-sm">
    <q-btn :disable="isJobRunning" label="保存" color="primary" type="submit" :loading="submitting"></q-btn>
    <q-btn :disable="isJobRunning" label="重置" title="放弃未保存的修改" @click="resetForm"></q-btn>
    <export-setting-btn-dialog />
    <span v-if="isJobRunning" class="text-caption text-grey-7">任务运行中，停止后才能保存</span>
  </div>
</template>

<script setup>
import { resetForm, submitting } from 'pages/settings/use-settings';
import ExportSettingBtnDialog from 'pages/settings/BtnDialogExportSetting';
import { isJobRunning } from 'src/store/systemState';
</script>
