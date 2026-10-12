<script setup lang="ts">
import { reactive, ref } from "vue";
import { ApiError, api } from "../api";
import { PRIORITIES, type NewTicket } from "../types";

const emit = defineEmits<{ created: [] }>();

const empty = (): NewTicket => ({ title: "", description: "", priority: "medium" });
const form = reactive<NewTicket>(empty());
const errors = ref<Record<string, string>>({});
const message = ref<string | null>(null);
const saving = ref(false);

async function submit() {
  saving.value = true;
  errors.value = {};
  message.value = null;
  try {
    await api.createTicket(form);
    Object.assign(form, empty());
    emit("created");
  } catch (err) {
    if (err instanceof ApiError) errors.value = err.fields;
    message.value = err instanceof Error ? err.message : "Could not create ticket";
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <form class="create" novalidate @submit.prevent="submit">
    <label>
      Title
      <input v-model="form.title" maxlength="200" required :aria-invalid="Boolean(errors.title)" />
      <span v-if="errors.title" class="field-error">Title {{ errors.title }}</span>
    </label>
    <label>
      Priority
      <select v-model="form.priority">
        <option v-for="p in PRIORITIES" :key="p" :value="p">{{ p }}</option>
      </select>
    </label>
    <label class="full">
      Description (optional)
      <textarea v-model="form.description" />
    </label>
    <div class="full">
      <button type="submit" :disabled="saving || !form.title.trim()">
        {{ saving ? "Creating…" : "Create ticket" }}
      </button>
      <div v-if="message && Object.keys(errors).length === 0" class="alert">{{ message }}</div>
    </div>
  </form>
</template>
