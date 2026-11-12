<template>
  <div>
    <q-list dense class="settings-list">
      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>任务通知</q-item-label>
          <q-item-label caption>字幕下载成功或任务失败时发送消息，发送在后台进行，不影响下载</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.enable" />
        </q-item-section>
      </q-item>

      <template v-if="form.enable">
        <q-item tag="label" v-ripple>
          <q-item-section>
            <q-item-label>下载成功时通知</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-toggle v-model="form.on_success" />
          </q-item-section>
        </q-item>
        <q-item tag="label" v-ripple>
          <q-item-section>
            <q-item-label>失败或未找到字幕时通知</q-item-label>
            <q-item-label caption>同一个视频 24 小时内只提醒一次</q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-toggle v-model="form.on_failure" />
          </q-item-section>
        </q-item>
        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>合并间隔（分钟）</q-item-label>
            <q-item-label caption>
              第一条立即发送，之后这段时间内的结果合并成一条摘要，避免大批量扫描时刷屏；0 为不合并
            </q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.number="form.digest_minutes"
              type="number"
              min="0"
              max="1440"
              class="settings-field"
              standout
              dense
              hide-bottom-space
              :rules="[(val) => (Number.isInteger(val) && val >= 0 && val <= 1440) || '请填写 0 到 1440 的整数']"
            />
          </q-item-section>
        </q-item>

        <q-separator spaced inset />

        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>Telegram Bot Token</q-item-label>
            <q-item-label caption>向 @BotFather 申请机器人后得到，形如 123456:ABC…</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.trim="form.telegram_bot_token"
              class="settings-field--wide"
              standout
              dense
              hide-bottom-space
            />
          </q-item-section>
        </q-item>
        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>Telegram Chat ID</q-item-label>
            <q-item-label caption>接收消息的用户、群组或频道 ID；要先给机器人发过一条消息</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.trim="form.telegram_chat_id"
              class="settings-field--wide"
              standout
              dense
              hide-bottom-space
              :rules="[(val) => !form.telegram_bot_token || !!val || '填写了 Bot Token 就要填写 Chat ID']"
            />
          </q-item-section>
        </q-item>
        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>Telegram API 地址（可选）</q-item-label>
            <q-item-label caption>
              留空使用 api.telegram.org，并走“进阶配置”里的代理（如已开启）；填写反代地址则直连
            </q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.trim="form.telegram_api_base"
              class="settings-field--wide"
              placeholder="https://api.telegram.org"
              standout
              dense
              hide-bottom-space
              :rules="[urlRule]"
            />
          </q-item-section>
        </q-item>

        <q-separator spaced inset />

        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>Bark 推送地址</q-item-label>
            <q-item-label caption>Bark App 首页显示的地址，只保留到设备 Key 为止</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.trim="form.bark_url"
              class="settings-field--wide"
              placeholder="https://api.day.app/你的Key"
              standout
              dense
              hide-bottom-space
              :rules="[urlRule]"
            />
          </q-item-section>
        </q-item>

        <q-separator spaced inset />

        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>通用 Webhook</q-item-label>
            <q-item-label caption>
              以 POST 发送 JSON：title、body、success、failed、events（每个任务的视频、结果、字幕源、错误）
            </q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-input
              v-model.trim="form.webhook_url"
              class="settings-field--wide"
              placeholder="http://192.168.1.10:8080/hook"
              standout
              dense
              hide-bottom-space
              :rules="[urlRule]"
            />
          </q-item-section>
        </q-item>

        <q-item>
          <q-item-section>
            <div>
              <q-btn label="发送测试通知" color="primary" outline :loading="testing" @click="sendTest" />
              <span class="text-caption text-grey q-ml-sm">用当前填写的内容发送，不需要先保存</span>
            </div>
            <div v-for="r in testResults" :key="r.channel" class="q-mt-xs text-caption">
              <q-badge :color="r.ok ? 'positive' : 'negative'" class="q-mr-xs">{{ r.ok ? '成功' : '失败' }}</q-badge>
              {{ CHANNEL_NAMES[r.channel] ?? r.channel }}
              <span v-if="r.error" class="text-negative" style="word-break: break-all">：{{ r.error }}</span>
            </div>
          </q-item-section>
        </q-item>
      </template>

      <q-separator spaced inset />

      <q-item>
        <q-item-section>
          <q-item-label>Jellyfin 入库后自动下载字幕</q-item-label>
          <q-item-label caption>
            在 Jellyfin 安装 Webhook 插件，添加一个 Generic Destination：Webhook Url 填下面的地址，Notification Type
            只勾选 Item Added，Item Type 勾选 Movies 和 Episodes，并勾选 Send All Properties。需要“媒体库设置”已连接
            Jellyfin 且目录映射正确。
          </q-item-label>
          <div v-if="apiKey.enabled && apiKey.key" class="row items-center no-wrap q-mt-sm">
            <code class="webhook-url col">{{ jellyfinWebhookUrl }}</code>
            <copy-to-clipboard-btn :text="jellyfinWebhookUrl" title="复制" />
          </div>
          <div v-else class="text-caption text-warning q-mt-sm">
            请先在“实验室”里开启 API key，这个地址需要它来鉴权。
          </div>
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import { toRefs } from '@vueuse/core';
import { formModel } from 'pages/settings/use-settings';
import HistoryApi from 'src/api/HistoryApi';
import CopyToClipboardBtn from 'components/CopyToClipboardBtn';
import { SystemMessage } from 'src/utils/message';

const CHANNEL_NAMES = { webhook: '通用 Webhook', telegram: 'Telegram', bark: 'Bark' };

const { notification_settings: form } = toRefs(formModel);

const apiKey = computed(() => formModel.experimental_function?.api_key_settings ?? {});
// Jellyfin must reach CSF at this address; the browser's address is the best guess we have.
const jellyfinWebhookUrl = computed(
  () => `${window.location.origin}/api/v1/jellyfin-webhook?api_key=${encodeURIComponent(apiKey.value.key)}`
);

const urlRule = (val) => !val || /^https?:\/\/\S+$/.test(val) || '地址要以 http:// 或 https:// 开头';

const testing = ref(false);
const testResults = ref([]);
const sendTest = async () => {
  testing.value = true;
  testResults.value = [];
  const [res, err] = await HistoryApi.testNotify(form.value);
  testing.value = false;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  testResults.value = res.results;
};
</script>

<style lang="scss" scoped>
.webhook-url {
  min-width: 0;
  padding: 6px 8px;
  border-radius: 4px;
  background: rgba(128, 128, 128, 0.15);
  word-break: break-all;
  font-size: 12px;
}
</style>
