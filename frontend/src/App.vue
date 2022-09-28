<template>
  <Nav />
  <div class="flex">
    <ChartGauge class="w-1/3" />
    <LineChart class="w-2/3" />
  </div>
</template>

<script setup>
  import Nav from './components/Nav.vue'
import ChartGauge from './components/ChartGauge.vue'
import LineChart from './components/LineChart.vue'
</script>
