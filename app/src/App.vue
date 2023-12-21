<template>
  <v-app>
    <v-main>
      <div class="container">
        <v-card width="80vw" max-width="640px" elevation="6">
          <v-data-table :headers="headers" :items="items">
            <template v-slot:body.prepend>
              <tr>
                <td><v-text-field v-model="newFrom" placeholder="http://exmaple.com/" density="compact" variant="outlined"
                    hide-details /></td>
                <td><v-text-field v-model="newTo" placeholder="http://exmaple.com/" density="compact" variant="outlined" hide-details />
                </td>
                <td>
                  <v-btn icon variant="text" size="x-small" @click="add">
                    <IconPlus />
                  </v-btn>
                </td>
              </tr>
            </template>
            <template v-slot:item.actions="{ item }">
              <v-btn icon variant="text" size="x-small" @click="del(item.id)">
                <IconTrash />
              </v-btn>
            </template>
          </v-data-table>
        </v-card>
      </div>
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { IconPlus, IconTrash } from '@tabler/icons-vue';

interface Dispatcher {
  id: number
  from: string
  to: string
}

const headers = [
  {
    title: 'from base url',
    key: 'from',
    width: "calc(50% - 32px)",
  },
  {
    title: 'to base url',
    key: 'to',
    width: "calc(50% - 32px)",
  },
  {
    title: 'actions',
    key: 'actions',
    width: "64px",
  }
];

const items = ref<Dispatcher[]>([]);

const newFrom = ref("");
const newTo = ref("");

async function load() {
  let res = await fetch('/list');
  let data = await res.json();
  items.value = data;
}

async function add() {
  let fromURL = new URL(newFrom.value);
  let toURL = new URL(newTo.value);
  let res = await fetch('/list', {
    method: 'POST',
    body: JSON.stringify({
      id: 0,
      from: fromURL.toString(),
      to: toURL.toString(),
    })
  });
  await load();
}

async function del(id: number) {
  let res = await fetch(`/list?id=${id}`, {
    method: 'DELETE',
  });
  await load();
}

onMounted(() => {
  load();
})
</script>

<style>
html {
  overflow: hidden;
}

.container {
  display: flex;
  width: 100vw;
  height: 100vh;
  align-items: center;
  justify-content: center;
  background-color: rgba(127 127 127 / 0.1);
}
</style>
