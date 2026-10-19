<script setup lang="ts">
import AdminPanel from './AdminPanel.vue'
import AdminApiKeysDocs from './AdminApiKeysDocs.vue'
</script>

<template>
  <!-- flat：让文档页右侧目录的 position:sticky 不被面板 overflow:hidden 裁掉 -->
  <AdminPanel flat title="API 文档" desc="接口使用说明、请求参数与返回体格式、多语言示例">
    <AdminApiKeysDocs />
  </AdminPanel>
</template>
