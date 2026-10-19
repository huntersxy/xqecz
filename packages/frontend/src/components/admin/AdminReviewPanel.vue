<script setup lang="ts">
// 审核与认领合并页（用户要求）：待审内容在上、认领申请在下。
// 两个列表各自加载、各自分页、各自处理，共用一个页签与一个 URL（?tab=review）。
// 旧链接 ?tab=pending / ?tab=claims 由 useAdminTabQuery 的 TAB_ALIASES 规范化到这里。
</script>

<template>
  <div class="admin-review">
    <AdminContentList mode="pending" />
    <AdminClaimList />
  </div>
</template>

<style lang="scss" scoped>
.admin-review {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-lg, 16px);
}
</style>
