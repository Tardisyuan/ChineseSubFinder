<template>
  <q-page class="q-pa-md history-page">
    <template v-if="stats">
      <div class="row q-col-gutter-md">
        <div v-for="card in cards" :key="card.label" class="col-6 col-md-3">
          <q-card flat bordered class="q-pa-md full-height">
            <div class="row items-center no-wrap text-caption text-grey">
              <q-icon :name="card.icon" size="18px" class="q-mr-xs" />
              {{ card.label }}
            </div>
            <div class="text-h5 q-mt-xs">{{ card.value }}</div>
            <div v-if="card.hint" class="text-caption text-grey">{{ card.hint }}</div>
          </q-card>
        </div>
      </div>

      <div class="row q-col-gutter-md q-mt-none">
        <div class="col-12 col-md-7">
          <q-card flat bordered class="full-height">
            <q-card-section class="row items-center q-gutter-x-md">
              <div class="text-subtitle1">每日结果</div>
              <q-space />
              <div v-for="r in RESULTS" :key="r.value" class="row items-center text-caption text-grey">
                <span class="legend-dot q-mr-xs" :class="`bg-${r.color}`" />{{ r.label }}
              </div>
            </q-card-section>
            <q-separator />
            <div class="day-chart q-pa-md">
              <div v-for="d in stats.days" :key="d.date" class="day-col">
                <div class="day-total text-grey">{{ dayTotal(d) || '' }}</div>
                <div class="day-bar">
                  <div
                    v-for="r in RESULTS"
                    :key="r.value"
                    :class="`bg-${r.color}`"
                    :style="{ height: `${(d[r.value] / maxDayTotal) * 100}%` }"
                  />
                </div>
                <div class="day-label text-grey">{{ d.date.slice(8) }}</div>
                <q-tooltip> {{ d.date }}：成功 {{ d.success }}，未找到 {{ d.no_sub }}，失败 {{ d.failed }} </q-tooltip>
              </div>
            </div>
          </q-card>
        </div>

        <div class="col-12 col-md-5">
          <q-card flat bordered class="full-height">
            <q-card-section class="row items-center">
              <div class="text-subtitle1">字幕源</div>
              <q-space />
              <div class="text-caption text-grey">按全部 {{ stats.total }} 条记录统计</div>
            </q-card-section>
            <q-separator />
            <q-markup-table v-if="stats.sites.length" flat dense>
              <thead>
                <tr>
                  <th class="text-left">名称</th>
                  <th class="text-right" title="该字幕源提供了候选字幕的任务数">有结果</th>
                  <th class="text-right" title="有结果的任务数 / 全部任务数">命中率</th>
                  <th class="text-right" title="最终保存的是该字幕源的字幕">被选用</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in stats.sites" :key="s.site">
                  <td>{{ s.site }}</td>
                  <td class="text-right">{{ s.offered }}</td>
                  <td class="text-right">{{ percent(s.offered, stats.total) }}</td>
                  <td class="text-right">{{ s.chosen }}</td>
                </tr>
              </tbody>
            </q-markup-table>
            <empty-state v-else icon="hourglass_empty" title="暂无数据" hint="有任务下载到字幕后这里才有统计" />
          </q-card>
        </div>
      </div>
    </template>

    <q-card flat bordered :class="{ 'q-mt-md': !!stats }">
      <q-tabs v-model="tab" dense no-caps align="left" active-color="primary" indicator-color="primary">
        <q-tab name="list" label="历史记录" />
        <q-tab name="missing" :label="`仍缺字幕${stats ? `（${stats.missing_count}）` : ''}`" />
      </q-tabs>
      <q-separator />

      <template v-if="tab === 'list'">
        <div class="row items-center q-pa-md q-gutter-sm">
          <q-select
            v-model="filter.result"
            :options="resultOptions"
            label="结果"
            emit-value
            map-options
            outlined
            dense
            class="filter-field"
            @update:model-value="reload"
          />
          <q-select
            v-model="filter.site"
            :options="siteOptions"
            label="字幕源"
            emit-value
            map-options
            outlined
            dense
            class="filter-field"
            @update:model-value="reload"
          />
          <q-input
            v-model="filter.from"
            type="date"
            label="从"
            stack-label
            outlined
            dense
            class="filter-field"
            @update:model-value="reload"
          />
          <q-input
            v-model="filter.to"
            type="date"
            label="到"
            stack-label
            outlined
            dense
            class="filter-field"
            @update:model-value="reload"
          />
          <q-input
            v-model="filter.keyword"
            class="col-grow col-md-auto"
            label="名称或路径"
            debounce="400"
            outlined
            dense
            clearable
            @update:model-value="reload"
          />
          <q-space class="gt-sm" />
          <q-btn flat dense round icon="refresh" title="刷新" :loading="loading" @click="refresh" />
        </div>
        <q-separator />
        <q-table
          v-model:pagination="pagination"
          :columns="columns"
          :rows="rows"
          :loading="loading"
          :rows-per-page-options="[20, 50, 100]"
          row-key="id"
          flat
          @request="onRequest"
        >
          <template v-slot:body-cell-result="{ row }">
            <q-td>
              <q-badge :color="RESULT_MAP[row.result]?.color ?? 'grey'">
                {{ RESULT_MAP[row.result]?.label ?? row.result }}
              </q-badge>
            </q-td>
          </template>
          <template v-slot:body-cell-title="{ row }">
            <q-td :title="row.video_f_path">
              <div class="ellipsis cell-title">{{ row.title }}</div>
            </q-td>
          </template>
          <template v-slot:body-cell-detail="{ row }">
            <q-td :title="row.result === 'success' ? row.sub_name : row.error">
              <div class="ellipsis cell-detail" :class="{ 'text-grey': row.result !== 'success' }">
                {{ row.result === 'success' ? row.sub_name : row.error }}
              </div>
            </q-td>
          </template>
          <template v-slot:no-data>
            <div class="full-width">
              <empty-state v-if="loading" loading title="正在加载…" />
              <empty-state v-else-if="loadError" icon="error_outline" title="历史加载失败" :hint="loadError">
                <q-btn color="primary" outline label="重试" @click="refresh" />
              </empty-state>
              <empty-state v-else-if="hasFilter" icon="filter_alt_off" title="没有符合筛选条件的记录" />
              <empty-state
                v-else
                icon="history"
                title="还没有下载记录"
                hint="每个字幕下载任务结束后会在这里留下一条记录"
              />
            </div>
          </template>
        </q-table>
      </template>

      <template v-else>
        <q-markup-table v-if="stats?.missing.length" flat dense>
          <thead>
            <tr>
              <th class="text-left">视频</th>
              <th class="text-right">尝试次数</th>
              <th class="text-left">最近一次</th>
              <th class="text-left">原因</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in stats.missing" :key="m.video_f_path">
              <td :title="m.video_f_path">
                <div class="ellipsis cell-title">{{ m.title }}</div>
              </td>
              <td class="text-right">{{ m.attempts }}</td>
              <td>{{ formatTime(m.last_time) }}</td>
              <td :title="m.last_error">
                <div class="ellipsis cell-detail text-grey">
                  {{ m.last_result === 'no_sub' ? '未找到字幕' : m.last_error }}
                </div>
              </td>
            </tr>
          </tbody>
        </q-markup-table>
        <empty-state
          v-else
          icon="task_alt"
          title="没有缺字幕的视频"
          hint="这里列出最近一次任务没有下载到字幕、且文件仍然存在的视频"
        />
        <div v-if="stats && stats.missing_count > stats.missing.length" class="q-pa-sm text-caption text-grey">
          只显示最近的 {{ stats.missing.length }} 个
        </div>
      </template>
    </q-card>
  </q-page>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import HistoryApi from 'src/api/HistoryApi';
import EmptyState from 'components/EmptyState';
import { SystemMessage } from 'src/utils/message';

const STAT_DAYS = 14;

const RESULTS = [
  { value: 'success', label: '成功', color: 'positive' },
  { value: 'no_sub', label: '未找到字幕', color: 'warning' },
  { value: 'failed', label: '失败', color: 'negative' },
];
const RESULT_MAP = Object.fromEntries(RESULTS.map((r) => [r.value, r]));

const pad = (n) => `${n}`.padStart(2, '0');
// the backend sends RFC 3339, possibly in UTC; show local time
const formatTime = (val) => {
  const d = new Date(val);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
};
const percent = (part, total) => (total > 0 ? `${Math.round((part / total) * 100)}%` : '-');

const tab = ref('list');
const stats = ref(null);
const rows = ref([]);
const loading = ref(false);
const loadError = ref('');
const filter = reactive({ result: '', site: '', from: '', to: '', keyword: '' });
const pagination = ref({ page: 1, rowsPerPage: 20, rowsNumber: 0 });

const columns = [
  { label: '时间', field: 'time', format: formatTime, align: 'left' },
  { label: '结果', field: 'result', name: 'result', align: 'left' },
  { label: '视频', field: 'title', name: 'title', align: 'left' },
  { label: '字幕源', field: 'site', align: 'left' },
  { label: '字幕 / 错误', field: 'sub_name', name: 'detail', align: 'left' },
];

const resultOptions = [{ value: '', label: '全部' }, ...RESULTS];
const siteOptions = computed(() => [
  { value: '', label: '全部' },
  ...(stats.value?.sites ?? []).filter((s) => s.chosen > 0).map((s) => ({ value: s.site, label: s.site })),
]);
const hasFilter = computed(() => Object.values(filter).some((v) => !!v));

const dayTotal = (d) => d.success + d.no_sub + d.failed;
const maxDayTotal = computed(() => Math.max(1, ...(stats.value?.days ?? []).map(dayTotal)));

const cards = computed(() => {
  const s = stats.value;
  const recent = s.days.reduce((sum, d) => sum + d.success, 0);
  return [
    { label: '任务记录', icon: 'history', value: s.total, hint: '最多保留 5000 条、180 天' },
    { label: '成功率', icon: 'task_alt', value: percent(s.success, s.total), hint: `成功 ${s.success} 次` },
    { label: `近 ${STAT_DAYS} 天下载字幕`, icon: 'subtitles', value: recent },
    { label: '仍缺字幕的视频', icon: 'subtitles_off', value: s.missing_count },
  ];
});

const loadStats = async () => {
  const [res, err] = await HistoryApi.getStats({ days: STAT_DAYS });
  if (err !== null) {
    SystemMessage.error(err.message);
    return;
  }
  stats.value = res;
};

const loadList = async (page = pagination.value.page, pageSize = pagination.value.rowsPerPage) => {
  loading.value = true;
  const params = Object.fromEntries(Object.entries(filter).filter(([, v]) => !!v));
  const [res, err] = await HistoryApi.getList({ ...params, page, page_size: pageSize });
  loading.value = false;
  if (err !== null) {
    loadError.value = err.message;
    rows.value = [];
    return;
  }
  loadError.value = '';
  rows.value = res.items;
  pagination.value = { page, rowsPerPage: pageSize, rowsNumber: res.total };
};

const onRequest = ({ pagination: p }) => loadList(p.page, p.rowsPerPage);
const reload = () => loadList(1);
const refresh = () => {
  loadStats();
  loadList();
};

onMounted(refresh);
</script>

<style lang="scss" scoped>
.history-page {
  .legend-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .day-chart {
    display: flex;
    align-items: flex-end;
    gap: 6px;
    height: 200px;
  }
  .day-col {
    flex: 1 1 0;
    min-width: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .day-total,
  .day-label {
    font-size: 11px;
    line-height: 16px;
    height: 16px;
    white-space: nowrap;
  }
  .day-bar {
    flex: 1;
    width: 100%;
    max-width: 28px;
    display: flex;
    flex-direction: column-reverse; // success at the bottom
    border-radius: 3px;
    overflow: hidden;
    background: rgba(128, 128, 128, 0.12);
  }
  .filter-field {
    width: 140px;
  }
  .cell-title {
    max-width: 280px;
  }
  .cell-detail {
    max-width: 360px;
  }
  @media (max-width: 599px) {
    .filter-field {
      width: calc(50% - 8px);
    }
    .cell-title,
    .cell-detail {
      max-width: 200px;
    }
  }
}
</style>
