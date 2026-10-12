<script setup lang="ts">
import { STATUSES, STATUS_LABELS, type Status, type Ticket } from "../types";

defineProps<{ tickets: Ticket[]; savingId: number | null }>();
const emit = defineEmits<{ statusChange: [id: number, status: Status] }>();

const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });

function onChange(id: number, event: Event) {
  emit("statusChange", id, (event.target as HTMLSelectElement).value as Status);
}
</script>

<template>
  <p v-if="tickets.length === 0" class="empty">No tickets here yet.</p>
  <table v-else>
    <thead>
      <tr>
        <th>Title</th>
        <th>Priority</th>
        <th>Status</th>
        <th>Created</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="ticket in tickets" :key="ticket.id">
        <td class="title">
          {{ ticket.title }}
          <small v-if="ticket.description">{{ ticket.description }}</small>
        </td>
        <td>
          <span :class="['badge', `priority-${ticket.priority}`]">{{ ticket.priority }}</span>
        </td>
        <td>
          <select
            :class="`status-${ticket.status}`"
            :value="ticket.status"
            :disabled="savingId === ticket.id"
            :aria-label="`Status of ${ticket.title}`"
            @change="onChange(ticket.id, $event)"
          >
            <option v-for="s in STATUSES" :key="s" :value="s">{{ STATUS_LABELS[s] }}</option>
          </select>
        </td>
        <td class="nowrap">{{ dateFormat.format(new Date(ticket.created_at)) }}</td>
      </tr>
    </tbody>
  </table>
</template>
