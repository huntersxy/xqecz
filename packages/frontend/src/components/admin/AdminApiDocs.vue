<script setup lang="ts">
import AdminPanel from './AdminPanel.vue'
import AdminApiKeysDocs from './AdminApiKeysDocs.vue'
</script>

<template>
  <AdminPanel title="API 文档" desc="接口使用说明、请求参数与返回体格式、多语言示例">
    <AdminApiKeysDocs />
  </AdminPanel>
</template>
