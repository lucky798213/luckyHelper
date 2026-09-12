<template>
  <div class="composer-wrap">
    <div class="composer">
      <textarea ref="ta" v-model="text" rows="1"
        placeholder="随心输入，Enter 发送 / Shift+Enter 换行"
        @keydown="onKey" @input="resize"></textarea>

      <div class="composer-foot">
        <!-- 上传文件（与侧栏同一条上传链路） -->
        <button class="icon-pill" type="button" title="上传文件" @click="fileInput.click()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
            <path d="M12 5v14M5 12h14" />
          </svg>
        </button>
        <input ref="fileInput" type="file" accept=".txt,.md,.pdf" multiple style="display:none" @change="onPick" />

        <!-- 知识库开关 -->
        <div class="pill" :class="{ on: chat.ragOn }" :title="chat.modeHint.text" @click="chat.toggleRag()">
          <span class="dot"></span>
          <span>知识库</span>
        </div>

        <!-- Skill 广场 -->
        <div class="pill" :class="{ on: skills.enabledCount > 0 }" title="Skill 广场" @click="$emit('open-skills')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round">
            <path d="M9 3h6v3.2a2 2 0 1 0 0 3.6V13H9v-3.2a2 2 0 1 0 0-3.6V3z" />
            <path d="M9 13H5.5a1.5 1.5 0 0 0 0 3H9" />
          </svg>
          <span>Skill</span>
          <span class="pill-count" v-if="skills.enabledCount">{{ skills.enabledCount }}</span>
        </div>

        <div class="composer-right">
          <span class="composer-hint">{{ chat.modeHint.text }}</span>
          <span class="model-tag" :title="modelTitle">
            <span class="dot" :class="{ mock: chat.isMock }"></span>{{ chat.model || '模型未上报' }}
          </span>
          <button class="send-btn" :class="{ stop: chat.loading }"
                  :disabled="!chat.loading && !text.trim()"
                  :title="chat.loading ? '停止生成' : '发送'"
                  @click="onSend">
            <svg v-if="chat.loading" viewBox="0 0 24 24" fill="currentColor"><rect x="7" y="7" width="10" height="10" rx="1.5" /></svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                 stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 19V5" /><path d="M5.5 11.5L12 5l6.5 6.5" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { useChat } from '../stores/chat'
import { useSkills } from '../stores/skills'
import { useDocs } from '../stores/docs'

defineEmits(['open-skills'])

const chat = useChat()
const skills = useSkills()
const docs = useDocs()

const text = ref('')
const ta = ref(null)
const fileInput = ref(null)

const modelTitle = computed(() => {
  if (!chat.model) return '未获取到后端模型信息'
  return chat.isMock ? `${chat.model}（mock 模式：未配置真实 API Key）` : chat.model
})

function resize() {
  const el = ta.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 180) + 'px'
}
function onKey(e) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); onSend() }
}
async function onSend() {
  if (chat.loading) { chat.stop(); return }
  const msg = text.value.trim()
  if (!msg) return
  text.value = ''
  await nextTick(); resize()
  chat.send(msg)
}
async function onPick(e) {
  for (const f of e.target.files) await docs.uploadFile(f)
  e.target.value = ''
}
</script>
