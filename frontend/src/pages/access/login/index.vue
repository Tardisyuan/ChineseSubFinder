<template>
  <q-layout view="lHh Lpr fff">
    <q-page-container>
      <q-page class="window-height window-width row justify-center items-center">
        <login-bg-area />
        <q-form @submit="submit" class="column q-pa-md">
          <div class="row">
            <q-card class="shadow-24" style="width: 400px; max-width: calc(100vw - 48px)">
              <q-card-section class="text-center text-white q-py-lg" style="background: var(--csf-drawer-bg)">
                <img src="icons/logo.png" alt="" style="filter: invert(100%); height: 48px" />
                <div class="text-h6">ChineseSubFinder</div>
                <div class="text-caption text-grey-5">系统登录</div>
              </q-card-section>
              <q-card-section>
                <div class="q-px-sm q-pt-md">
                  <q-input
                    v-model="form.username"
                    autofocus
                    autocomplete="username"
                    lazy-rules
                    :rules="[(val) => !!val || '用户名不能为空']"
                    label="用户名"
                  >
                    <template v-slot:prepend>
                      <q-icon name="person" />
                    </template>
                  </q-input>
                  <q-input
                    v-model="form.password"
                    type="password"
                    autocomplete="current-password"
                    lazy-rules
                    :rules="[(val) => !!val || '密码不能为空']"
                    label="密码"
                  >
                    <template v-slot:prepend>
                      <q-icon name="lock" />
                    </template>
                  </q-input>
                </div>
              </q-card-section>

              <q-card-actions class="q-px-lg q-pb-lg">
                <q-btn
                  unelevated
                  size="lg"
                  color="primary"
                  type="submit"
                  :loading="submitting"
                  class="full-width text-white"
                  label="登录"
                />
              </q-card-actions>
            </q-card>
          </div>
        </q-form>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { SystemMessage } from 'src/utils/message';
import { useRouter } from 'vue-router';
import AccessApi from 'src/api/AccessApi';
import { LocalStorage } from 'quasar';
import { userState } from 'src/store/userState';
import LoginBgArea from 'pages/access/login/LoginBgArea';

const router = useRouter();
const form = reactive({
  username: '',
  password: '',
});

const submitting = ref(false);

const submit = async () => {
  submitting.value = true;
  const formData = { ...form };
  delete formData.confirmPassword;
  const [res, err] = await AccessApi.login(formData);
  submitting.value = false;
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  const userData = {
    accessToken: res.access_token,
    username: form.username,
  };
  Object.assign(userState, userData);
  LocalStorage.set('token', userData);
  router.push('/');
};
</script>
