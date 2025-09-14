<template>
  <div style="position: relative; height: 100%; width: 100%;">
    <Line ref="chartRef" :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { ref } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  TimeScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(
  CategoryScale,
  LinearScale,
  TimeScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

export default {
  name: 'PlayerCountChart',
  components: {
    Line
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