<script setup lang="ts">
import { STATUSES, STATUS_LABELS, type Stats } from "../types";

defineProps<{ stats: Stats | null }>();
</script>

<template>
  <div class="stats" aria-label="Tickets by status">
    <span v-for="s in STATUSES" :key="s" class="stat">
      <strong>{{ stats ? stats[s] : "–" }}</strong>{{ STATUS_LABELS[s] }}
    </span>
  </div>
</template>
