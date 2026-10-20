<script setup lang="ts">
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
