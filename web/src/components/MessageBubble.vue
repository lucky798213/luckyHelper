<template>
  <div class="msg-row" :class="msg.role">
    <!-- 用户消息：右上浅灰块 -->
    <div v-if="msg.role === 'user'" class="u-msg">{{ msg.text }}</div>

    <!-- 兼容旧格式（存的是 html 串） -->
    <div v-else-if="msg.html" class="ai-msg">
      <div class="answer-text" v-html="msg.html"></div>
    </div>

    <!-- AI 消息：通栏排版 -->
    <div v-else class="ai-msg">
      <div v-if="msg.memory" class="memory-note">
        <span class="memory-tag">记忆</span>
        <span class="memory-content">{{ msg.memory }}</span>
      </div>

      <ThinkPanel :msg="msg" />

      <div v-if="msg.interrupted" class="interrupted-badge">已中断</div>

      <div v-if="!msg.answer && msg.streaming" class="typing"><span></span><span></span><span></span></div>
      <div v-else class="answer-text" v-html="rendered"></div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '../utils/markdown'
import ThinkPanel from './ThinkPanel.vue'

const props = defineProps({ msg: { type: Object, required: true } })
const rendered = computed(() => renderMarkdown(props.msg.answer || ''))
</script>
