<template>
  <q-btn
    flat
    dense
    icon="bug_report"
    :label="$q.screen.gt.xs ? '问题反馈' : undefined"
    title="跳转到Github issue页面"
    @click="gotoGithubIssuePage"
  />
</template>

<script setup>
import { gotoGithubIssuePage } from 'src/utils/common';
</script>
