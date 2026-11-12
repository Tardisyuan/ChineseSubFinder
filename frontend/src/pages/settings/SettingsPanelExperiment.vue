<template>
  <div>
    <q-list class="settings-list" dense>
      <q-item>
        <q-item-section>
          <q-item-label>自动转换字幕文件编码</q-item-label>
          <q-item-label caption>自动转换到目标编码，如果不是特殊情况，不建议开启，仅对新下载字幕生效</q-item-label>
          <q-item v-if="form.auto_change_sub_encode.enable">
            <q-item-section avatar top>
              <q-radio
                v-for="(v, k) in DESC_ENCODE_TYPE_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.auto_change_sub_encode.des_encode_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.auto_change_sub_encode.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item tag="label" :disable="!isChsChtChangerEnable" v-ripple>
        <q-item-section>
          <q-item-label>简、繁字幕互转功能</q-item-label>
          <q-item-label caption
            >需要开启"自动转换字幕文件编码"功能，并设置为转码"UTF-8"，否则无法启用和生效</q-item-label
          >
          <q-item v-if="form.chs_cht_changer.enable">
            <q-item-section avatar top>
              <q-radio
                :disable="!isChsChtChangerEnable"
                v-for="(v, k) in AUTO_CONVERT_LANG_NAME_MAP"
                :key="k"
                :label="v"
                v-model="form.chs_cht_changer.des_chinese_language_type"
                :val="~~k"
              />
            </q-item-section>
          </q-item>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle :disable="!isChsChtChangerEnable" v-model="form.chs_cht_changer.enable" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>API key</q-item-label>
          <q-item-label caption>
            本程序提供一些接口给开发者使用，通过API key鉴权，具体参见
            <!-- eslint-disable -->
            <a
              href="https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/ApiKey%E8%AE%BE%E8%AE%A1/ApiKey%E8%AE%BE%E8%AE%A1.md"
              class="text-primary"
              target="_blank"
              >开发文档</a
            >
          </q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.api_key_settings.enabled" />
        </q-item-section>
      </q-item>

      <template v-if="form.api_key_settings.enabled">
        <q-item>
          <q-btn label="重新生成密钥" color="primary" size="sm" @click="generateApiKey" />
        </q-item>
        <q-item class="q-mt-sm">
          <q-item-section>
            <q-input
              v-model="form.api_key_settings.key"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
              readonly
            >
              <template #append>
                <copy-to-clipboard-btn v-if="form.api_key_settings.key" :text="form.api_key_settings.key" size="sm" />
              </template>
            </q-input>
          </q-item-section>
        </q-item>
      </template>
    </q-list>
  </div>
</template>

<script setup>
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import {
  AUTO_CONVERT_LANG_NAME_MAP,
  DESC_ENCODE_TYPE_NAME_MAP,
  DESC_ENCODE_TYPE_UTF8,
} from 'src/constants/SettingConstants';
import { computed } from 'vue';
import CopyToClipboardBtn from 'components/CopyToClipboardBtn';

const { experimental_function: form } = toRefs(formModel);

const isChsChtChangerEnable = computed(
  () =>
    formModel.experimental_function.auto_change_sub_encode?.enable &&
    formModel.experimental_function.auto_change_sub_encode?.des_encode_type === DESC_ENCODE_TYPE_UTF8
);

const generateUuid = () =>
  'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    // eslint-disable-next-line no-bitwise
    const r = (Math.random() * 16) | 0;
    // eslint-disable-next-line no-bitwise
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });

const generateApiKey = () => {
  const uuid = generateUuid();
  formModel.experimental_function.api_key_settings.key = uuid;
};
</script>
