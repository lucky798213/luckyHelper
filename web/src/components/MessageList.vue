<template>
  <div class="messages" ref="scroller">
    <!-- 空态：Codex 式居中开场 -->
    <div v-if="!sessions.messages.length" class="welcome">
      <div class="welcome-mark">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"
             stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 3l2.2 5.4L20 10.5l-5.8 2.1L12 18l-2.2-5.4L4 10.5l5.8-2.1L12 3z" />
          <path d="M18.5 16.5l.8 2 2 .8-2 .8-.8 2-.8-2-2-.8 2-.8.8-2z" />
        </svg>
      </div>
      <h1 class="welcome-title">今天想做点什么？</h1>
      <p class="welcome-desc">
        开启「知识库」可检索你上传的文件；在 Skill 广场装好工具后，我会自动推理并调用它们完成任务。
      </p>
      <div class="hint-chips">
        <button v-for="q in QUICK" :key="q" class="hint-chip" type="button" @click="ask(q)">{{ q }}</button>
      </div>
    </div>

    <MessageBubble v-for="(m, i) in sessions.messages" :key="i" :msg="m" />
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useSessions } from '../stores/sessions'
import { useChat } from '../stores/chat'
import MessageBubble from './MessageBubble.vue'

const sessions = useSessions()
const chat = useChat()
const scroller = ref(null)

const QUICK = [
  '你是谁？',
  '现在几点？',
  '北京天气怎么样？',
  '记住：我喜欢周杰伦的音乐',
  '写一份关于人工智能发展趋势的报告',
]

function ask(q) { chat.send(q) }

// 消息变化（含流式增量）时滚到底
watch(() => JSON.stringify(sessions.messages), async () => {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}, { flush: 'post' })
</script>
