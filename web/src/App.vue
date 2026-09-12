<template>
  <SideBar :collapsed="collapsed" @toggle-collapse="collapsed = !collapsed" />

  <div class="main">
    <button v-if="collapsed" class="side-toggle-float" title="展开侧栏" @click="collapsed = false">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
           stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <path d="M9 4v16" />
      </svg>
    </button>

    <MessageList />
    <ChatInput @open-skills="skillHubOpen = true" />
  </div>

  <AuthModal v-if="auth.overlay" />
  <SkillHub v-if="skillHubOpen" @close="skillHubOpen = false" />
  <DocViewer />
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { setUnauthorizedHandler } from './api/client'
import { useAuth } from './stores/auth'
import { useDocs } from './stores/docs'
import { useSkills } from './stores/skills'
import { useSessions } from './stores/sessions'
import { useChat } from './stores/chat'
import SideBar from './components/SideBar.vue'
import MessageList from './components/MessageList.vue'
import ChatInput from './components/ChatInput.vue'
import AuthModal from './components/AuthModal.vue'
import SkillHub from './components/SkillHub.vue'
import DocViewer from './components/DocViewer.vue'

const auth = useAuth()
const docs = useDocs()
const skills = useSkills()
const sessions = useSessions()
const chat = useChat()
const skillHubOpen = ref(false)
const collapsed = ref(localStorage.getItem('luckyhelper_sidebar_collapsed') === '1')

watch(collapsed, (v) => localStorage.setItem('luckyhelper_sidebar_collapsed', v ? '1' : '0'))

// 401 → 清 token + 弹登录层 + 中断在飞的对话
setUnauthorizedHandler(() => {
  chat.abortInflight()
  auth.logout()
})

function initApp() {
  docs.loadLibrary()
  skills.loadInstalled()
  chat.loadStatus()
}

watch(() => auth.loggedIn, (v) => {
  if (v) initApp()
  else { sessions.reset(); docs.reset(); skills.reset() }
})

onMounted(() => { if (auth.loggedIn) initApp() })
</script>
