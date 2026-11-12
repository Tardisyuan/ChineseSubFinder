<template>
  <div>
    <q-list class="settings-list" dense>
      <q-item-label header>字幕站点</q-item-label>

      <q-item v-for="item in activeSuppliers" :key="item.name">
        <q-item-section>
          <q-item-label class="text-bold">{{ item.name }}</q-item-label>
          <q-item-label caption style="word-break: break-all">{{ item.root_url }}</q-item-label>
          <q-item-label caption>每日下载次数限制：{{ dailyLimitText(item.daily_download_limit) }}</q-item-label>
        </q-item-section>
        <q-item-section side>
          <edit-sub-source-btn-dialog :data="item" @update="(data) => handleSubSourceUpdate(item, data)" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item-label header>Assrt API</q-item-label>

      <q-item tag="label">
        <q-item-section>
          <q-item-label>启用 Assrt</q-item-label>
          <q-item-label caption>
            <div>
              需要先
              <a href="https://assrt.net/user/register.xml" target="_blank" class="text-primary">注册</a>
              ，然后在
              <a href="https://assrt.net/usercp.php" target="_blank" class="text-primary">用户面板</a>
              获取 Token（<a href="https://assrt.net/api/doc" target="_blank" class="text-primary">API 文档</a>）
            </div>
            <ul class="q-pl-md q-my-xs">
              <li>一般用户是 5c次/min 的 API 请求限制</li>
              <li>建议设置完 Token 后，重启程序或者容器！</li>
              <li>搜索字幕效果未知，如果不用就关闭即可</li>
              <li>建议配合“保存多字幕”的选项服用（如果你使用 Emby 的话）</li>
            </ul>
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.assrt_settings.enabled" />
        </q-item-section>
      </q-item>

      <q-item v-if="form.assrt_settings.enabled">
        <q-item-section>
          <q-input
            v-model="form.assrt_settings.token"
            placeholder="填写你的API Token"
            label="Assrt API Token"
            standout
            dense
            :rules="[(val) => !!val || '启用 Assrt 时必须填写 Token']"
          />
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import { computed } from 'vue';
import EditSubSourceBtnDialog from 'pages/settings/BtnDialogEditSubSource';

const { subtitle_sources: form } = toRefs(formModel);

// Sites that are gone or no longer wired in; their settings stay in the config but are not shown.
const DEAD_SUPPLIERS = ['a4k', 'zimuku', 'subtitle_best', 'csf', 'subhd'];
const activeSuppliers = computed(() =>
  Object.values(formModel.advanced_settings.suppliers_settings ?? {}).filter((s) => !DEAD_SUPPLIERS.includes(s.name))
);

const dailyLimitText = (limit) => {
  if (limit === -1) return '不限制';
  if (limit === 0) return '已禁用';
  return `${limit} 次`;
};

const handleSubSourceUpdate = (item, data) => {
  formModel.advanced_settings.suppliers_settings[item.name].root_url = data.url;
  formModel.advanced_settings.suppliers_settings[item.name].daily_download_limit = data.dailyLimit;
};
</script>
