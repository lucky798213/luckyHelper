<template>
  <aside class="sidebar" :class="{ collapsed }">
    <!-- 品牌 + 操作 -->
    <div class="side-head">
      <div class="brand">
        <span class="brand-mark">A</span>
        <span class="brand-name">luckyHelper</span>
      </div>
      <button class="icon-btn" title="搜索对话" @click="toggleSearch">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <circle cx="11" cy="11" r="7" /><path d="M20 20l-3.6-3.6" />
        </svg>
      </button>
      <button class="icon-btn" title="收起侧栏" @click="$emit('toggle-collapse')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
             stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="4" width="18" height="16" rx="2" /><path d="M9 4v16" /><path d="M16 9.5l-2.5 2.5L16 14.5" />
        </svg>
      </button>
    </div>

    <!-- 搜索（按标题过滤近期对话） -->
    <div v-if="searchOpen" class="search-row">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <circle cx="11" cy="11" r="7" /><path d="M20 20l-3.6-3.6" />
      </svg>
      <input ref="searchInput" v-model="query" placeholder="搜索对话标题" />
      <button v-if="query" class="icon-btn tiny" title="清空" @click="query = ''">×</button>
    </div>

    <!-- 新建对话 -->
    <div class="new-chat" @click="newChat">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <path d="M12 5v14M5 12h14" />
      </svg>
      <span>新对话</span>
      <span class="spacer"></span>
      <svg class="compose" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"
           stroke-linecap="round" stroke-linejoin="round">
        <path d="M4 20h4L19 9a2.1 2.1 0 0 0-3-3L5 17v3z" /><path d="M14.5 6.5L17.5 9.5" />
      </svg>
    </div>

    <!-- 个人文件（可折叠） -->
    <div class="sec-head" @click="filesOpen = !filesOpen">
      <span class="chev" :class="{ open: filesOpen }">▶</span>
      <span>个人文件</span>
      <span class="spacer"></span>
      <span class="sec-count" v-if="docs.uploaded.length">{{ docs.uploaded.length }}</span>
    </div>
    <div class="sec-body" v-show="filesOpen">
      <div class="upload-zone" :class="{ drag: dragging }"
           @click="fileInput.click()"
           @dragover.prevent="dragging = true"
           @dragleave="dragging = false"
           @drop.prevent="onDrop">
        <div class="uz-icon">↑</div>
        <div class="uz-text">上传个人文件</div>
        <div class="uz-sub">点击或拖拽 · txt / md / pdf</div>
      </div>
      <input ref="fileInput" type="file" accept=".txt,.md,.pdf" multiple style="display:none" @change="onPick" />
      <div class="doc-list">
        <div v-if="!docs.uploaded.length" class="doc-empty">暂无上传文件</div>
        <div v-for="d in docs.uploaded.slice(0, 6)" :key="d.docHash || d.name" class="doc-item">
          <span class="doc-icon" :title="d.needsOCR ? '需要 OCR' : '已解析'">{{ d.needsOCR ? '!' : '·' }}</span>
          <span class="doc-name" :title="docTitle(d)">{{ d.name }}</span>
          <span class="doc-chunks">{{ d.needsOCR ? '需 OCR' : `${d.chunks || 0} 块` }}</span>
          <span v-if="d.docHash" class="doc-del" title="删除" @click.stop="removeDoc(d)">×</span>
        </div>
      </div>
    </div>

    <!-- 本地文档库（可折叠） -->
    <div class="sec-head" @click="libraryOpen = !libraryOpen">
      <span class="chev" :class="{ open: libraryOpen }">▶</span>
      <span>本地文档库</span>
      <span class="spacer"></span>
      <button class="sec-action" title="刷新" @click.stop="docs.loadLibrary()">刷新</button>
      <span class="sec-count" v-if="docs.library.length">{{ docs.library.length }}</span>
    </div>
    <div class="sec-body library-panel" v-show="libraryOpen">
      <div class="doc-list">
        <div v-if="!docs.library.length" class="doc-empty">暂无本地文档。让 Agent 生成报告并保存后，会出现在这里。</div>
        <div v-for="d in docs.library.slice(0, 20)" :key="d.id" class="doc-item"
             @click="docs.openViewer(d.id)">
          <span class="doc-icon">§</span>
          <span class="doc-name" :title="d.title || ''">{{ d.title || '未命名文档' }}</span>
          <span class="doc-chunks">v{{ d.latest_version || 0 }}</span>
          <span class="doc-ingest" title="重新入库 RAG" @click.stop="docs.ingest(d.id)">↻</span>
        </div>
      </div>
    </div>

    <!-- 近期对话 -->
    <div class="sec-head" style="cursor:default">
      <span>近期对话</span>
    </div>
    <div class="recent">
      <div v-if="!filteredSessions.length" class="sess-empty">
        {{ query ? '没有匹配的对话' : '暂无对话记录' }}
      </div>
      <div v-for="s in filteredSessions" :key="s.id"
           class="session-item" :class="{ active: s.id === sessions.currentId }"
           @click="switchTo(s.id)">
        <div class="sess-info">
          <div class="sess-title" :title="s.title">{{ s.title }}</div>
        </div>
        <span class="sess-meta">{{ fmtDate(s.ts) }}</span>
        <span class="sess-del" title="删除" @click.stop="del(s.id)">✕</span>
      </div>
    </div>

    <div v-if="auth.loggedIn" class="user-bar">
      <span class="avatar-mini">{{ (auth.username || '?').slice(0, 1) }}</span>
      <span class="uname" :title="auth.username">{{ auth.username }}</span>
      <button class="logout-btn" type="button" @click="logout">登出</button>
    </div>
  </aside>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { useDocs } from '../stores/docs'
import { useSessions } from '../stores/sessions'
import { useAuth } from '../stores/auth'
import { useChat } from '../stores/chat'

defineProps({ collapsed: { type: Boolean, default: false } })
defineEmits(['toggle-collapse'])

const docs = useDocs()
const sessions = useSessions()
const auth = useAuth()
const chat = useChat()

const fileInput = ref(null)
const searchInput = ref(null)
const dragging = ref(false)
const searchOpen = ref(false)
const query = ref('')

// 两个资料分组默认折叠，只露出「近期对话」——贴近极简侧栏
const filesOpen = ref(false)
const libraryOpen = ref(false)

const filteredSessions = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return sessions.sessions
  return sessions.sessions.filter(s => (s.title || '').toLowerCase().includes(q))
})

function toggleSearch() {
  searchOpen.value = !searchOpen.value
  query.value = ''
  if (searchOpen.value) nextTick(() => searchInput.value && searchInput.value.focus())
}

async function onPick(e) {
  for (const f of e.target.files) await docs.uploadFile(f)
  e.target.value = ''
}
async function onDrop(e) {
  dragging.value = false
  for (const f of e.dataTransfer.files) await docs.uploadFile(f)
}
function docTitle(d) {
  return [d.name, d.pages ? d.pages + ' 页' : '', d.textChars ? d.textChars + ' 字' : '', d.parser || '']
    .filter(Boolean).join(' · ')
}
async function removeDoc(d) { if (confirm('确定删除「' + d.name + '」？')) await docs.deleteDoc(d.docHash) }
function fmtDate(ts) { return new Date(ts).toLocaleDateString('zh', { month: 'short', day: 'numeric' }) }
function switchTo(id) { chat.abortInflight(); sessions.switchSession(id) }
function del(id) { sessions.deleteSession(id) }
function newChat() { chat.abortInflight(); sessions.newSession() }
function logout() { chat.abortInflight(); auth.logout() }
</script>
