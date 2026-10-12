<script setup lang="ts">
import { ref, watch } from "vue";
import { api } from "./api";
import StatsBar from "./components/StatsBar.vue";
import TicketForm from "./components/TicketForm.vue";
import TicketTable from "./components/TicketTable.vue";
import { STATUSES, STATUS_LABELS, type Page, type Stats, type Status, type Ticket } from "./types";

const PAGE_SIZE = 10;

const filter = ref<Status | undefined>();
const page = ref(1);
const result = ref<Page<Ticket> | null>(null);
const stats = ref<Stats | null>(null);
const error = ref<string | null>(null);
const savingId = ref<number | null>(null);

async function load() {
  try {
    const [list, counts] = await Promise.all([
      api.listTickets({ status: filter.value, page: page.value, limit: PAGE_SIZE }),
      api.stats(),
    ]);
    // A status change can empty the last page of a filtered view; step back instead.
    if (list.data.length === 0 && page.value > 1) {
      page.value -= 1;
      return;
    }
    result.value = list;
    stats.value = counts;
    error.value = null;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not load tickets";
  }
}

watch([filter, page], load, { immediate: true });

function selectFilter(next: Status | undefined) {
  filter.value = next;
  page.value = 1;
}

function onCreated() {
  // New tickets appear on page one, newest first.
  if (page.value === 1) void load();
  else page.value = 1;
}

async function changeStatus(id: number, status: Status) {
  savingId.value = id;
  try {
    await api.updateTicket(id, { status });
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Could not update ticket";
  } finally {
    savingId.value = null;
  }
}
</script>

<template>
  <main class="page">
    <header class="page-header">
      <h1>Ticket Tracker</h1>
      <StatsBar :stats="stats" />
    </header>

    <section class="card">
      <h2>New ticket</h2>
      <TicketForm @created="onCreated" />
    </section>

    <section class="card">
      <div class="toolbar">
        <h2>Tickets</h2>
        <div class="filters" role="group" aria-label="Filter by status">
          <button :aria-pressed="filter === undefined" @click="selectFilter(undefined)">All</button>
          <button v-for="s in STATUSES" :key="s" :aria-pressed="filter === s" @click="selectFilter(s)">
            {{ STATUS_LABELS[s] }}
          </button>
        </div>
      </div>

      <div v-if="error" class="alert">{{ error }}</div>
      <TicketTable
        v-if="result"
        :tickets="result.data"
        :saving-id="savingId"
        @status-change="changeStatus"
      />

      <nav v-if="result && result.pagination.total_pages > 1" class="pager" aria-label="Pagination">
        <button class="secondary" :disabled="page <= 1" @click="page--">Previous</button>
        <span>
          Page {{ result.pagination.page }} of {{ result.pagination.total_pages }}
          ({{ result.pagination.total }} tickets)
        </span>
        <button class="secondary" :disabled="page >= result.pagination.total_pages" @click="page++">
          Next
        </button>
      </nav>
    </section>
  </main>
</template>
