<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const canvasRef = ref<HTMLCanvasElement | null>(null)
const animationId = ref<number | null>(null)
const audioContext = ref<AudioContext | null>(null)
const analyser = ref<AnalyserNode | null>(null)
const dataArray = ref<Uint8Array | null>(null)
const isListening = ref(false)
const hasPermission = ref(false)
const showPrompt = ref(true)
const rotationAngle = ref(0)

let mediaStream: MediaStream | null = null

async function startAudioVisualization() {
  try {
    const canvas = canvasRef.value
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    function resizeCanvas() {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight
    }
    resizeCanvas()
    window.addEventListener('resize', resizeCanvas)

    audioContext.value = new (window.AudioContext || (window as any).webkitAudioContext)()
    analyser.value = audioContext.value.createAnalyser()
    analyser.value.fftSize = 256
    
    const bufferLength = analyser.value.frequencyBinCount
    dataArray.value = new Uint8Array(bufferLength)

    try {
      mediaStream = await navigator.mediaDevices.getDisplayMedia({ 
        video: { width: 1, height: 1 }, 
        audio: true 
      })
      const source = audioContext.value.createMediaStreamSource(mediaStream)
      source.connect(analyser.value)
      isListening.value = true
      hasPermission.value = true
      showPrompt.value = false
    } catch (err) {
      console.log('无法获取系统音频权限，尝试麦克风')
      try {
        mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true })
        const source = audioContext.value.createMediaStreamSource(mediaStream)
        source.connect(analyser.value)
        isListening.value = true
        hasPermission.value = true
        showPrompt.value = false
      } catch (micErr) {
        console.log('无法获取麦克风权限，使用模拟信号')
        const oscillator = audioContext.value.createOscillator()
        const gain = audioContext.value.createGain()
        gain.gain.value = 0.1
        oscillator.connect(gain)
        gain.connect(analyser.value)
        oscillator.start()
        isListening.value = true
        hasPermission.value = true
        showPrompt.value = false
      }
    }

    function draw() {
      animationId.value = requestAnimationFrame(draw)
      
      if (!analyser.value || !dataArray.value || !ctx || !canvas) return
      
      analyser.value.getByteFrequencyData(dataArray.value as any)
      
      const width = canvas.width
      const height = canvas.height
      const centerX = width / 2
      const centerY = height / 2
      
      ctx.clearRect(0, 0, width, height)

      const average = (dataArray.value as unknown as number[]).reduce((a, b) => a + b, 0) / dataArray.value.length
      const intensity = average / 255
      const time = Date.now() * 0.001
      
      const baseRadius = 150
      const waveRadius = baseRadius + intensity * 150
      
      // 根据响度控制旋转速度（越响越快）
      const rotationSpeed = 0.002 + intensity * 0.015
      rotationAngle.value += rotationSpeed
      
      // 基础波动值：仅在整体有音频时生效，确保无声时完全静默
      const baseAmplitude = intensity > 0.02 ? 15 : 0
      
      const segments = 60 // 每段 60 个点，三段共 180 个点
      const arcAngle = (Math.PI * 2) / 3 // 每段 120 度
      const angleStep = arcAngle / segments
      
      // 绘制三段圆弧，每段都是完整的 0-100% 频率
      for (let arc = 0; arc < 3; arc++) {
        const arcStartAngle = rotationAngle.value + arc * arcAngle
        
        // 外层光晕
        for (let layer = 3; layer >= 1; layer--) {
          const layerRadius = waveRadius + layer * 40
          const layerAlpha = 0.15 / layer
          
          ctx.beginPath()
          for (let i = 0; i <= segments; i++) {
            const angle = arcStartAngle + i * angleStep
            const dataIndex = Math.floor((i / segments) * dataArray.value.length)
            const value = dataArray.value[dataIndex] / 255
            
            const waveAmplitude = value * 100 + baseAmplitude
            const radius = layerRadius + waveAmplitude * Math.sin(angle * 6 + time * 2)
            const x = centerX + Math.cos(angle) * radius
            const y = centerY + Math.sin(angle) * radius
            
            if (i === 0) ctx.moveTo(x, y)
            else ctx.lineTo(x, y)
          }
          ctx.closePath()
          ctx.strokeStyle = `rgba(255, 255, 255, ${layerAlpha + intensity * 0.1})`
          ctx.lineWidth = 2 + layer * 2
          ctx.stroke()
        }
        
        // 主圆环
        ctx.beginPath()
        for (let i = 0; i <= segments; i++) {
          const angle = arcStartAngle + i * angleStep
          const dataIndex = Math.floor((i / segments) * dataArray.value.length)
          const value = dataArray.value[dataIndex] / 255
          
          const wave1 = Math.sin(angle * 12 + time * 3) * 0.5
          const wave2 = Math.sin(angle * 8 - time * 2) * 0.3
          const wave3 = Math.sin(angle * 16 + time * 4) * 0.2
          
          const radius = waveRadius + (value * 180 + baseAmplitude) * (wave1 + wave2 + wave3 + 1)
          
          const x = centerX + Math.cos(angle) * radius
          const y = centerY + Math.sin(angle) * radius
          
          if (i === 0) ctx.moveTo(x, y)
          else ctx.lineTo(x, y)
        }
        ctx.closePath()
        
        const mainGradient = ctx.createRadialGradient(centerX, centerY, baseRadius * 0.3, centerX, centerY, waveRadius + 100)
        mainGradient.addColorStop(0, 'rgba(255, 255, 255, 1)')
        mainGradient.addColorStop(0.6, 'rgba(240, 240, 255, 0.95)')
        mainGradient.addColorStop(1, 'rgba(220, 220, 240, 0.8)')
        
        ctx.strokeStyle = mainGradient
        ctx.lineWidth = 8
        ctx.shadowColor = 'rgba(255, 255, 255, 0.8)'
        ctx.shadowBlur = 30 + intensity * 40
        ctx.stroke()
        ctx.shadowBlur = 0
      }
      
      const coreRadius = baseRadius * 0.6
      const coreGradient = ctx.createRadialGradient(centerX, centerY, 0, centerX, centerY, coreRadius)
      coreGradient.addColorStop(0, `rgba(255, 255, 255, ${0.9 + intensity * 0.1})`)
      coreGradient.addColorStop(0.4, `rgba(255, 255, 255, ${0.6 + intensity * 0.2})`)
      coreGradient.addColorStop(0.7, `rgba(240, 240, 255, ${0.3 + intensity * 0.2})`)
      coreGradient.addColorStop(1, 'rgba(255, 255, 255, 0)')
      
      ctx.beginPath()
      ctx.arc(centerX, centerY, coreRadius + intensity * 30, 0, Math.PI * 2)
      ctx.fillStyle = coreGradient
      ctx.fill()
      
      const particleCount = 60
      for (let i = 0; i < particleCount; i++) {
        const angle = (i / particleCount) * Math.PI * 2 + time * 0.5
        const dataIndex = Math.floor((i / particleCount) * dataArray.value.length)
        const value = dataArray.value[dataIndex] / 255
        
        const particleRadius = waveRadius + 50 + value * 100
        const x = centerX + Math.cos(angle) * particleRadius
        const y = centerY + Math.sin(angle) * particleRadius
        
        const size = 1 + value * 3
        const alpha = 0.3 + value * 0.7
        
        ctx.beginPath()
        ctx.arc(x, y, size, 0, Math.PI * 2)
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha})`
        ctx.fill()
      }

      ctx.fillStyle = `rgba(255, 255, 255, ${0.9 + intensity * 0.1})`
      ctx.font = '18px -apple-system, BlinkMacSystemFont, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.shadowColor = 'rgba(255, 255, 255, 0.5)'
      ctx.shadowBlur = 10
      ctx.fillText(isListening.value ? '聆听中' : '沉浸音乐', centerX, centerY)
      ctx.shadowBlur = 0
    }

    draw()
  } catch (error) {
    console.error('音频可视化初始化失败:', error)
  }
}

function stopAudioVisualization() {
  if (animationId.value) {
    cancelAnimationFrame(animationId.value)
  }
  
  if (mediaStream) {
    mediaStream.getTracks().forEach(track => track.stop())
  }
  
  if (audioContext.value) {
    audioContext.value.close()
  }
}

onMounted(() => {
  document.body.style.overflow = 'hidden'
  document.documentElement.style.overflow = 'hidden'
})

onUnmounted(() => {
  document.body.style.overflow = ''
  document.documentElement.style.overflow = ''
  stopAudioVisualization()
})
</script>

<template>
  <div class="relative w-full h-screen overflow-hidden">
    <canvas ref="canvasRef" class="absolute inset-0 w-full h-full z-0 pointer-events-none" />
    
    <div v-if="showPrompt" class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-6">
      <button 
        @click="startAudioVisualization"
        class="px-10 py-5 bg-white/10 backdrop-blur-xl border border-white/30 rounded-full text-white text-xl font-light tracking-wider hover:bg-white/20 hover:scale-105 transition-all duration-500 shadow-[0_8px_40px_rgba(0,0,0,0.4)] cursor-pointer">
        开启音频沉浸
      </button>
      <p class="text-white/50 text-sm max-w-xs text-center leading-relaxed">
        请在弹窗中选择 <span class="text-white/80 font-medium">整个屏幕</span>，并勾选底部 <span class="text-white/80 font-medium">分享系统音频</span>
      </p>
    </div>
  </div>
</template>
