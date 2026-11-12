export const JOB_STATUS_PENDING = 0;
export const JOB_STATUS_IN_PROGRESS = 1;
export const JOB_STATUS_FAILED = 2;
export const JOB_STATUS_COMPLETED = 3;
export const JOB_STATUS_DOWNLOADING = 4;
export const JOB_STATUS_IGNORE = 5;

export const JOB_STATUS_MAP = {
  [JOB_STATUS_PENDING]: '等待下载',
  [JOB_STATUS_IN_PROGRESS]: '处理中',
  [JOB_STATUS_FAILED]: '失败',
  [JOB_STATUS_COMPLETED]: '已完成',
  [JOB_STATUS_DOWNLOADING]: '下载中',
  [JOB_STATUS_IGNORE]: '忽略',
};

// Quasar colour names (for q-badge), picked so the white label stays readable
export const JOB_STATUS_COLOR_MAP = {
  [JOB_STATUS_PENDING]: 'grey-7',
  [JOB_STATUS_IN_PROGRESS]: 'blue-7',
  [JOB_STATUS_COMPLETED]: 'positive',
  [JOB_STATUS_FAILED]: 'negative',
  [JOB_STATUS_DOWNLOADING]: 'blue-7',
  [JOB_STATUS_IGNORE]: 'orange-9',
};

export const JOB_STATUS_OPTIONS = Object.keys(JOB_STATUS_MAP).map((k) => ({
  label: JOB_STATUS_MAP[k],
  value: parseInt(k, 10),
}));
