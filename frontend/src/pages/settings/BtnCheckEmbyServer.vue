<template>
  <q-btn label="检测连接" color="secondary" :loading="loading" @click="checkEmbyServer" />
</template>

<script setup>
import { ref } from 'vue';
import CommonApi from 'src/api/CommonApi';
import { SystemMessage } from 'src/utils/message';
import { formModel } from 'pages/settings/use-settings';

const loading = ref(false);

const checkEmbyServer = async () => {
  loading.value = true;
  const [res, err] = await CommonApi.checkEmbyServer({
    address_url: formModel.emby_settings.address_url,
    api_key: formModel.emby_settings.api_key,
    server_type: formModel.emby_settings.server_type,
  });
  if (err !== null) {
    SystemMessage.error(err.message);
  } else if (res.message !== 'ok') {
    SystemMessage.error(res.message);
  } else {
    SystemMessage.success(`${formModel.emby_settings.server_type === 'jellyfin' ? 'Jellyfin' : 'Emby'} 连接正常`);
  }
  loading.value = false;
};
</script>
