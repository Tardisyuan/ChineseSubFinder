<template>
  <div>
    <q-list dense class="settings-list">
      <q-item-label header>字幕挑选</q-item-label>

      <template v-if="form.sub_priority">
        <q-item>
          <q-item-section>
            <q-item-label caption>
              多个字幕可选时按这个顺序比较：先看版本是否匹配，再看语言偏好，然后是字幕格式，最后是站点顺序。
            </q-item-label>
          </q-item-section>
        </q-item>

        <q-item tag="label" v-ripple>
          <q-item-section>
            <q-item-label>优先匹配视频版本</q-item-label>
            <q-item-label caption>
              字幕文件名和视频文件名的发布组、片源（如 DSNP、WEB-DL）、分辨率越一致越优先，时间轴更不容易错位。
              视频文件名里没有这些信息时不起作用。
            </q-item-label>
          </q-item-section>
          <q-item-section avatar>
            <q-toggle v-model="form.sub_priority.match_release" />
          </q-item-section>
        </q-item>

        <q-item class="stack-xs">
          <q-item-section>
            <q-item-label>语言偏好</q-item-label>
          </q-item-section>
          <q-item-section side>
            <div class="row">
              <q-radio v-model="form.sub_priority.language" val="bilingual" label="双语优先" />
              <q-radio v-model="form.sub_priority.language" val="chs" label="简体优先" />
              <q-radio v-model="form.sub_priority.language" val="cht" label="繁体优先" />
            </div>
          </q-item-section>
        </q-item>
      </template>

      <q-item class="stack-xs">
        <q-item-section>
          <q-item-label>字幕格式优先级</q-item-label>
        </q-item-section>
        <q-item-section side>
          <div class="row">
            <q-radio
              v-for="(v, k) in SUB_TYPE_PRIORITY_NAME_MAP"
              :key="k"
              v-model="form.sub_type_priority"
              :val="~~k"
              :label="v"
            />
          </div>
        </q-item-section>
      </q-item>

      <q-item v-if="form.sub_priority" class="stack-xs">
        <q-item-section top>
          <q-item-label>站点顺序</q-item-label>
          <q-item-label caption>越靠上越优先，用箭头调整</q-item-label>
        </q-item-section>
        <q-item-section side>
          <q-list dense bordered separator class="settings-field rounded-borders">
            <q-item v-for="(site, i) in form.sub_priority.site_order" :key="site">
              <q-item-section avatar class="text-grey">{{ i + 1 }}</q-item-section>
              <q-item-section class="text-black">{{ site }}</q-item-section>
              <q-item-section side>
                <div class="row no-wrap">
                  <q-btn
                    flat
                    dense
                    round
                    size="sm"
                    icon="arrow_upward"
                    title="上移"
                    :disable="i === 0"
                    @click="moveSite(i, -1)"
                  />
                  <q-btn
                    flat
                    dense
                    round
                    size="sm"
                    icon="arrow_downward"
                    title="下移"
                    :disable="i === form.sub_priority.site_order.length - 1"
                    @click="moveSite(i, 1)"
                  />
                </div>
              </q-item-section>
            </q-item>
          </q-list>
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>字幕保存</q-item-label>

      <q-item>
        <q-item-section>
          <q-item-label>字幕文件的命名格式</q-item-label>
          <q-item v-for="(v, k) in SUB_NAME_FORMAT_NAME_MAP" :key="k" tag="label" v-ripple>
            <q-item-section avatar top>
              <q-radio v-model="form.sub_name_formatter" :val="~~k" />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ v }}</q-item-label>
              <q-item-label caption style="word-break: break-all">
                {{ subNameFormatDescMap[k] }}
              </q-item-label>
            </q-item-section>
          </q-item>
          <div class="text-caption text-negative">
            * 修改此选项后需要重启容器，启动阶段会对所有字幕做格式调整，耗时可能较长
          </div>
        </q-item-section>
      </q-item>

      <q-item v-if="SUB_NAME_FORMAT_EMBY === form.sub_name_formatter" tag="label" v-ripple>
        <q-item-section>
          <q-item-label>保存多字幕</q-item-label>
          <q-item-label caption>每个视频保存每个网站找到的最佳字幕，仅 Emby 格式可用</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.save_multi_sub" />
        </q-item-section>
      </q-item>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>保存整季的缓存字幕</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.save_full_season_tmp_subtitles" />
        </q-item-section>
      </q-item>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>自动校正字幕时间轴</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.fix_time_line" />
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>扫描范围</q-item-label>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>跳过中文电影</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.scan_logic.skip_chinese_movie" />
        </q-item-section>
      </q-item>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>跳过中文连续剧</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.scan_logic.skip_chinese_series" />
        </q-item-section>
      </q-item>

      <q-item class="stack-xs">
        <q-item-section top>
          <q-item-label>自定义视频扩展名</q-item-label>
          <q-item-label caption>
            原生支持 mp4、mkv、rmvb、iso、m2ts，其他扩展名在这里添加。要带点并用小写，例如 .ts
          </q-item-label>
        </q-item-section>
        <q-item-section side top>
          <div v-for="(item, i) in form.custom_video_exts" :key="i" class="row no-wrap items-start q-gutter-x-xs">
            <q-input
              v-model="form.custom_video_exts[i]"
              class="settings-field col"
              placeholder=".ts"
              standout
              dense
              :rules="[(val) => !!val || '扩展名不能为空']"
            />
            <q-btn
              class="q-mt-xs"
              icon="remove_circle_outline"
              color="negative"
              flat
              dense
              round
              title="删除"
              @click="form.custom_video_exts.splice(i, 1)"
            ></q-btn>
          </div>
          <q-btn
            class="self-start"
            icon="add"
            label="添加扩展名"
            color="primary"
            flat
            dense
            @click="form.custom_video_exts.push('')"
          ></q-btn>
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>网络</q-item-label>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>使用代理</q-item-label>
          <q-item-label caption>支持 HTTP 和 SOCKS5 代理</q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.proxy_settings.use_proxy" />
        </q-item-section>
      </q-item>

      <q-item v-if="form.proxy_settings.use_proxy" dense>
        <q-item-section>
          <div class="row q-col-gutter-sm">
            <q-select
              class="col-4 col-sm-2"
              v-model="form.proxy_settings.use_which_proxy_protocol"
              :options="Object.keys(PROXY_TYPE_NAME_MAP).map((e) => ({ label: PROXY_TYPE_NAME_MAP[e], value: e }))"
              label="协议"
              standout
              dense
              emit-value
              map-options
            />
            <q-input
              class="col-8 col-sm-4"
              v-model="form.proxy_settings.input_proxy_address"
              standout
              dense
              label="代理服务器"
            />
            <q-input
              class="col-6 col-sm-3"
              v-model="form.proxy_settings.input_proxy_port"
              standout
              dense
              label="代理端口"
            />
            <q-input
              class="col-6 col-sm-3"
              v-model="form.proxy_settings.local_http_proxy_server_port"
              standout
              dense
              label="本地端口"
            />
          </div>

          <div class="q-mt-xs row q-col-gutter-sm items-center">
            <div class="col-12 col-sm-auto">
              <q-checkbox v-model="form.proxy_settings.need_pwd" label="需要账号认证" />
            </div>
            <q-input
              class="col-6 col-sm"
              :disable="!form.proxy_settings.need_pwd"
              v-model="form.proxy_settings.input_proxy_username"
              standout
              dense
              label="账号"
            />
            <q-input
              class="col-6 col-sm"
              :disable="!form.proxy_settings.need_pwd"
              v-model="form.proxy_settings.input_proxy_password"
              standout
              dense
              label="密码"
            />
          </div>

          <div class="q-mt-sm">
            <proxy-check-btn
              :settings="form.proxy_settings"
              label="测试代理服务"
              size="md"
              icon="bolt"
              color="primary"
            />
          </div>
        </q-item-section>
      </q-item>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>启用 TMDB API</q-item-label>
          <q-item-label caption>用来获取影片的中文片名，部分字幕源要用中文片名搜索</q-item-label>
        </q-item-section>
        <q-item-section avatar top>
          <q-toggle v-model="form.tmdb_api_settings.enable" />
        </q-item-section>
      </q-item>

      <q-item v-if="form.tmdb_api_settings.enable">
        <q-item-section>
          <q-input
            v-model="form.tmdb_api_settings.api_key"
            standout
            dense
            label="TMDB ApiKey（v3）"
            hint="请使用 v3 的 ApiKey；国内访问 TMDB API 可能需要代理"
            :rules="[(val) => (form.tmdb_api_settings.enable && !!val) || '启用 TMDB API 时必须填写 ApiKey']"
          />
          <q-checkbox
            v-model="form.tmdb_api_settings.use_alternate_base_url"
            label="使用备用的 TMDB API 地址"
            title="如果连接不上 TMDB API，可以尝试勾选这个选项"
          />
          <div class="q-mt-xs">
            <btn-check-tmdb-api />
          </div>
        </q-item-section>
      </q-item>

      <q-separator spaced inset />

      <q-item-label header>下载队列</q-item-label>

      <q-item>
        <q-item-section>
          <div class="row q-col-gutter-x-md">
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.max_retry_times"
              type="number"
              label="最大重试次数"
              hint="单个任务失败超过这个次数后会降一级"
              standout
              dense
              :rules="[(val) => !!val || '不能为空']"
            />
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.one_job_time_out"
              type="number"
              label="单个任务的超时时间"
              standout
              dense
              suffix="秒"
              :rules="[(val) => !!val || '不能为空']"
            />
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.interval"
              type="number"
              label="下载任务之间的间隔"
              hint="防止频率太高触发防爬检测"
              standout
              dense
              suffix="秒"
              :rules="[(val) => !!val || '不能为空']"
            />
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.one_sub_download_interval"
              type="number"
              label="失败后重新下载的最小间隔"
              standout
              dense
              suffix="小时"
              :rules="[(val) => !!val || '不能为空']"
            />
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.expiration_time"
              type="number"
              label="下载时效"
              hint="视频创建超过这个天数就不再下载，标记为失败"
              standout
              dense
              suffix="天"
              :rules="[(val) => !!val || '不能为空']"
            />
            <q-input
              class="col-12 col-sm-6"
              v-model.number="form.task_queue.download_sub_during_x_days"
              type="number"
              label="有内置字幕的视频下载时效"
              hint="创建超过这个天数且有内置中文字幕的视频不再下载"
              standout
              dense
              suffix="天"
              :rules="[(val) => !!val || '不能为空']"
            />
          </div>
          <q-input
            class="q-mt-sm"
            v-model="form.task_queue.check_pulic_ip_target_site"
            label="检查公网 IP 的目标网站（可留空）"
            hint="内置站点失效后才需要填写。网站必须直接返回 IP 字符串，多个站点用英文分号 ; 分隔"
            standout
            dense
          />
        </q-item-section>
      </q-item>

      <q-item class="stack-xs q-mt-md">
        <q-item-section>
          <q-item-label>下载缓存过期时间</q-item-label>
        </q-item-section>
        <q-item-section side>
          <div class="row no-wrap q-gutter-x-xs settings-field">
            <q-input class="col" standout dense type="number" v-model.number="form.download_file_cache.ttl"> </q-input>
            <q-select
              standout
              dense
              :options="[
                { label: '小时', value: 'hour' },
                { label: '秒', value: 'second' },
              ]"
              emit-value
              map-options
              v-model.number="form.download_file_cache.unit"
            ></q-select>
          </div>
        </q-item-section>
      </q-item>

      <q-separator spaced inset></q-separator>

      <q-item-label header>调试</q-item-label>

      <q-item tag="label" v-ripple>
        <q-item-section>
          <q-item-label>调试模式</q-item-label>
          <q-item-label caption>输出 Debug 级别的日志，反馈问题时再打开</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-toggle v-model="form.debug_mode" />
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<script setup>
import {
  SUB_NAME_FORMAT_EMBY,
  SUB_NAME_FORMAT_NORMAL,
  SUB_NAME_FORMAT_NAME_MAP,
  SUB_TYPE_PRIORITY_NAME_MAP,
  PROXY_TYPE_NAME_MAP,
  SUB_NAME_VIDEO,
} from 'src/constants/SettingConstants';
import { formModel } from 'pages/settings/use-settings';
import { toRefs } from '@vueuse/core';
import ProxyCheckBtn from 'components/ProxyCheckBtn';
import BtnCheckTmdbApi from 'pages/settings/BtnCheckTmdbApi';

const subNameFormatDescMap = {
  [SUB_NAME_FORMAT_NORMAL]: '兼容性更好，AAA.zh.ass or AAA.zh.default.ass。',
  [SUB_NAME_FORMAT_EMBY]: 'AAA.chinese(简英,subhd).ass or AAA.chinese(简英,xunlei).default.ass。',
  [SUB_NAME_VIDEO]: '无语言描述后缀，AAA.ass or AAA.srt',
};

const { advanced_settings: form } = toRefs(formModel);

const moveSite = (i, delta) => {
  const order = form.value.sub_priority.site_order;
  const j = i + delta;
  [order[i], order[j]] = [order[j], order[i]];
};
</script>
