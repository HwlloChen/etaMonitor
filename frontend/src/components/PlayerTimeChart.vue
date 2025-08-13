<template>
  <div style="position: relative; height: 100%; width: 100%;">
    <Bar 
      ref="chartRef"
      :data="chartData" 
      :options="chartOptions"
    />
  </div>
</template>

<script>
import { ref } from 'vue'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
)

export default {
  name: 'PlayerTimeChart',
  components: {
    Bar
  },
  props: {
    chartData: {
      type: Object,
      required: true
    },
    chartOptions: {
      type: Object,
      required: true
    }
  },
  setup() {
    const chartRef = ref(null)
    
    return {
      chartRef
    }
  }
}
</script>