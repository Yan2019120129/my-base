<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

import { appConfig } from '@/config/env'

type CapabilityIcon = 'backend' | 'files' | 'delivery'

/** 能力卡片的展示数据。 */
interface CapabilityCard {
  index: string
  title: string
  description: string
  tags: string[]
  icon: CapabilityIcon
}

const navItems = [
  { id: 'focus', label: '能力' },
  { id: 'work', label: '系统' },
  { id: 'about', label: '原则' },
  { id: 'contact', label: '联系' },
]

const heroMeta = [
  { label: '01 / 专注', value: '后端与工具' },
  { label: '02 / 方法', value: '小而完整' },
  { label: '03 / 坐标', value: 'UTC + 8' },
]

const capabilityCards: CapabilityCard[] = [
  {
    index: '01',
    title: '可靠的后端',
    description: '用 Go 构建清晰的 API、后台任务与数据边界，让系统在高并发下依然可预测。',
    tags: ['Go', 'Gin', 'GORM', 'MySQL'],
    icon: 'backend',
  },
  {
    index: '02',
    title: '文件与内容',
    description: '上传、断点续传、存储、分享和权限控制，把文件流转做成真正顺手的工具。',
    tags: ['Tus', 'Storage', 'Share'],
    icon: 'files',
  },
  {
    index: '03',
    title: '产品化交付',
    description: '从数据模型到管理界面，再到部署与观测，关注每一个真正会被使用的细节。',
    tags: ['Admin', 'DX', 'Deploy'],
    icon: 'delivery',
  },
]

const showcaseItems = [
  { index: '01', title: '断点续传', description: '网络中断不再意味着从头开始。' },
  { index: '02', title: '可控分享', description: '有效期、撤销与权限，都在掌握之中。' },
  { index: '03', title: '清晰管理', description: '文件、分享与用户关系一眼可见。' },
]

const systemStats = [
  { label: 'FILES', value: '128' },
  { label: 'STORAGE', value: '4.8 GB' },
  { label: 'SHARES', value: '12' },
]

const principles = [
  {
    index: '01',
    title: '长期主义',
    description: '今天的选择要为下一次修改留出空间。能持续演进的系统，比一次性炫技更有价值。',
  },
  {
    index: '02',
    title: '清晰边界',
    description: '先定义问题，再选择工具。数据如何流动、责任落在哪里，比框架名字更重要。',
  },
  {
    index: '03',
    title: '细节有重量',
    description: '速度、反馈和可读性都是功能。用户感受到的每一秒，都是系统设计的一部分。',
  },
]

const tickerItems = ['GO', 'SYSTEMS', 'FILE FLOW', 'APIs', 'DETAILS', 'LONG-TERM']
const adminUrl = appConfig.adminUrl
const logoUrl = `${import.meta.env.BASE_URL}logo.jpeg`
const currentYear = new Date().getFullYear()

const rootRef = ref<HTMLElement | null>(null)
const navRef = ref<HTMLElement | null>(null)
const menuToggleRef = ref<HTMLButtonElement | null>(null)
const currentTime = ref('--:--')
const isScrolled = ref(false)
const isMenuOpen = ref(false)
const activeSection = ref('')

let revealObserver: IntersectionObserver | undefined
let clockTimer: number | undefined
const pointerCleanups: Array<() => void> = []

/** 更新页头滚动状态和当前可见章节。 */
function updateScrollState() {
  isScrolled.value = window.scrollY > 24

  const probeLine = window.innerHeight * 0.35
  let nextSection = ''

  navItems.forEach((item) => {
    const section = document.getElementById(item.id)
    if (section && section.getBoundingClientRect().top <= probeLine) {
      nextSection = item.id
    }
  })

  activeSection.value = nextSection
}

/** 切换移动端导航，可传入目标状态用于强制关闭。 */
function toggleMenu(forceOpen?: boolean) {
  isMenuOpen.value = typeof forceOpen === 'boolean' ? forceOpen : !isMenuOpen.value
}

/** 关闭移动端导航。 */
function closeMenu() {
  isMenuOpen.value = false
}

/** 点击导航链接后收起移动端菜单。 */
function handleNavigationClick(event: MouseEvent) {
  if (event.target instanceof Element && event.target.closest('a')) {
    closeMenu()
  }
}

/** 按 Escape 或点击导航与按钮外部时关闭菜单。 */
function handleDocumentInteraction(event: Event) {
  if (!isMenuOpen.value) {
    return
  }

  if (event instanceof KeyboardEvent && event.key === 'Escape') {
    closeMenu()
    menuToggleRef.value?.focus()
    return
  }

  if (
    event instanceof MouseEvent &&
    !navRef.value?.contains(event.target as Node) &&
    !menuToggleRef.value?.contains(event.target as Node)
  ) {
    closeMenu()
  }
}

/** 以上海时区刷新页头展示的当前时间。 */
function updateClock() {
  try {
    currentTime.value =
      new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).format(new Date()) + ' CST'
  } catch {
    currentTime.value = new Date().toTimeString().slice(0, 5)
  }
}

/** 为首次进入视口的内容添加淡入动画。 */
function bindRevealAnimation() {
  const items = Array.from(rootRef.value?.querySelectorAll<HTMLElement>('.reveal') ?? [])
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

  if (!('IntersectionObserver' in window) || prefersReducedMotion) {
    items.forEach((item) => item.classList.add('is-visible'))
    return
  }

  revealObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) {
          return
        }

        entry.target.classList.add('is-visible')
        revealObserver?.unobserve(entry.target)
      })
    },
    {
      rootMargin: '0px 0px -8% 0px',
      threshold: 0.12,
    },
  )

  items.forEach((item) => revealObserver?.observe(item))
}

/** 为展示卡片绑定跟随指针的细微高光。 */
function bindPointerEffects() {
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (prefersReducedMotion) {
    return
  }

  const surfaces = rootRef.value?.querySelectorAll<HTMLElement>('[data-tilt-surface]') ?? []

  surfaces.forEach((surface) => {
    const card = surface.matches('.hero-card, .system-frame')
      ? surface
      : surface.querySelector<HTMLElement>('.hero-card, .system-frame')

    if (!card) {
      return
    }

    const handlePointerMove = (event: PointerEvent) => {
      if (event.pointerType === 'touch') {
        return
      }

      const rect = surface.getBoundingClientRect()
      card.style.setProperty('--pointer-x', `${((event.clientX - rect.left) / rect.width) * 100}%`)
      card.style.setProperty('--pointer-y', `${((event.clientY - rect.top) / rect.height) * 100}%`)
    }

    const handlePointerLeave = () => {
      card.style.removeProperty('--pointer-x')
      card.style.removeProperty('--pointer-y')
    }

    surface.addEventListener('pointermove', handlePointerMove)
    surface.addEventListener('pointerleave', handlePointerLeave)
    pointerCleanups.push(() => {
      surface.removeEventListener('pointermove', handlePointerMove)
      surface.removeEventListener('pointerleave', handlePointerLeave)
    })
  })
}

onMounted(() => {
  updateScrollState()
  updateClock()
  bindRevealAnimation()
  bindPointerEffects()

  window.addEventListener('scroll', updateScrollState, { passive: true })
  window.addEventListener('resize', updateScrollState)
  document.addEventListener('keydown', handleDocumentInteraction)
  document.addEventListener('click', handleDocumentInteraction)
  clockTimer = window.setInterval(updateClock, 30_000)
})

onBeforeUnmount(() => {
  if (clockTimer !== undefined) {
    window.clearInterval(clockTimer)
  }

  revealObserver?.disconnect()
  pointerCleanups.forEach((cleanup) => cleanup())

  window.removeEventListener('scroll', updateScrollState)
  window.removeEventListener('resize', updateScrollState)
  document.removeEventListener('keydown', handleDocumentInteraction)
  document.removeEventListener('click', handleDocumentInteraction)
})
</script>

<template>
  <div ref="rootRef" class="home-page">
    <a class="skip-link" href="#main">跳到主要内容</a>

    <header class="site-header" :class="{ 'is-scrolled': isScrolled }">
      <a class="brand" href="#main" aria-label="020200 首页">
        <span class="brand-mark" aria-hidden="true">02</span>
        <span class="brand-copy">
          <strong>020200</strong>
          <small>DIGITAL PRACTICE</small>
        </span>
      </a>

      <button
        ref="menuToggleRef"
        class="menu-toggle"
        type="button"
        aria-controls="site-nav"
        :aria-expanded="isMenuOpen"
        @click="toggleMenu()"
      >
        <span></span>
        <span></span>
        <span class="sr-only">{{ isMenuOpen ? '关闭导航' : '打开导航' }}</span>
      </button>

      <nav
        id="site-nav"
        ref="navRef"
        class="site-nav"
        :class="{ 'is-open': isMenuOpen }"
        aria-label="主导航"
        @click="handleNavigationClick"
      >
        <a
          v-for="item in navItems"
          :key="item.id"
          :href="`#${item.id}`"
          :class="{ 'is-active': activeSection === item.id }"
          :aria-current="activeSection === item.id ? 'location' : undefined"
        >
          {{ item.label }}
        </a>
        <a class="mobile-admin-link" :href="adminUrl"
          >进入工作台 <span aria-hidden="true">↗</span></a
        >
      </nav>

      <a class="header-link" :href="adminUrl">
        <span>工作台</span>
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path
            d="M4 12 12 4M6 4h6v6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.4"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </a>
    </header>

    <main id="main">
      <section class="hero" aria-labelledby="hero-title">
        <div class="hero-copy">
          <p class="eyebrow reveal"><span class="status-dot"></span>GO / BACKEND / PRODUCT</p>
          <h1 class="reveal" id="hero-title">把复杂留给系统，<span>把简单留给使用。</span></h1>
          <p class="hero-intro reveal">
            020200 是我的数字工作台。这里构建 Go 服务、文件系统与克制的产品体验，
            也记录那些值得长期打磨的东西。
          </p>
          <div class="hero-actions reveal">
            <a class="button button-primary" href="#focus">
              <span>看看我在做什么</span>
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <path
                  d="M4 10h11M11 6l4 4-4 4"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.6"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </a>
            <a class="button button-ghost" href="#contact">聊聊想法</a>
          </div>
          <dl class="hero-meta reveal">
            <div v-for="item in heroMeta" :key="item.label">
              <dt>{{ item.label }}</dt>
              <dd>{{ item.value }}</dd>
            </div>
          </dl>
        </div>

        <div class="hero-visual reveal" data-tilt-surface>
          <figure class="hero-card">
            <div class="hero-card-head">
              <span>020200.CORE</span>
              <span class="live-status"><i></i>ONLINE</span>
            </div>
            <div class="hero-card-media">
              <div class="media-grid" aria-hidden="true"></div>
              <img :src="logoUrl" alt="一只正在写代码的 Go Gopher" width="700" height="700" />
              <span class="media-caption">BUILD / SHIP / LEARN</span>
            </div>
            <figcaption class="hero-card-foot">
              <div>
                <strong>Still coding.</strong>
                <span>把想法跑起来</span>
              </div>
              <time>{{ currentTime }}</time>
            </figcaption>
          </figure>

          <div class="floating-card floating-card-top" aria-hidden="true">
            <span>NOW</span>
            <strong>Build useful things</strong>
          </div>
          <div class="floating-card floating-card-bottom" aria-hidden="true">
            <span>LOOP</span>
            <strong>Ship / refine / repeat</strong>
          </div>
        </div>
      </section>

      <div class="ticker" aria-hidden="true">
        <div class="ticker-track">
          <div v-for="groupIndex in 2" :key="groupIndex" class="ticker-group">
            <template v-for="(item, index) in tickerItems" :key="`${groupIndex}-${item}`">
              <span>{{ item }}</span>
              <i v-if="index < tickerItems.length - 1"></i>
            </template>
          </div>
        </div>
      </div>

      <section id="focus" class="section capabilities" aria-labelledby="focus-title">
        <div class="section-head">
          <div>
            <p class="section-kicker reveal">01 — WHAT I DO</p>
            <h2 class="reveal" id="focus-title">从底层到界面，<br />做完整的事。</h2>
          </div>
          <p class="section-note reveal">
            不把技术当作终点。真正重要的是：问题被理解，系统能运行，体验足够自然。
          </p>
        </div>

        <div class="capability-grid">
          <article v-for="card in capabilityCards" :key="card.index" class="capability-card reveal">
            <div class="card-top">
              <span>{{ card.index }}</span>
              <span class="card-icon" aria-hidden="true">
                <svg v-if="card.icon === 'backend'" viewBox="0 0 32 32">
                  <rect
                    x="5"
                    y="7"
                    width="22"
                    height="18"
                    rx="2"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                  />
                  <path
                    d="m12 13-4 3 4 3M20 13l4 3-4 3M18 10l-4 12"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg v-else-if="card.icon === 'files'" viewBox="0 0 32 32">
                  <path
                    d="M7 10.5 16 5l9 5.5v11L16 27l-9-5.5v-11Z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linejoin="round"
                  />
                  <path
                    d="m7 10.5 9 5.5 9-5.5M16 16v11"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg v-else viewBox="0 0 32 32">
                  <path
                    d="M6 25V7h20v18H6Z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M6 12h20M11 17h6M11 21h10"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.4"
                    stroke-linecap="round"
                  />
                </svg>
              </span>
            </div>
            <h3>{{ card.title }}</h3>
            <p>{{ card.description }}</p>
            <ul class="tag-list" aria-label="技术标签">
              <li v-for="tag in card.tags" :key="tag">{{ tag }}</li>
            </ul>
          </article>
        </div>
      </section>

      <section id="work" class="showcase-wrap" aria-labelledby="work-title">
        <div class="showcase">
          <div class="showcase-copy">
            <p class="section-kicker reveal">02 — SELECTED SYSTEM</p>
            <h2 class="reveal" id="work-title">文件流转，<br />简单直接。</h2>
            <p class="showcase-intro reveal">
              一个以可靠传输为核心的轻量系统：大文件断点续传、可撤销分享链接、细粒度权限，
              以及足够清晰的后台操作。
            </p>
            <ul class="showcase-list">
              <li v-for="item in showcaseItems" :key="item.index" class="reveal">
                <span>{{ item.index }}</span>
                <div>
                  <strong>{{ item.title }}</strong>
                  <p>{{ item.description }}</p>
                </div>
              </li>
            </ul>
            <a class="text-link reveal" :href="adminUrl">
              <span>进入工作台</span>
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <path
                  d="M4 10h11M11 6l4 4-4 4"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </a>
          </div>

          <div class="system-frame reveal" data-tilt-surface>
            <div class="system-topbar">
              <div class="window-dots" aria-hidden="true"><i></i><i></i><i></i></div>
              <span>files.020200</span>
              <span>SECURE / LOCAL</span>
            </div>
            <div class="system-shell">
              <aside class="system-sidebar" aria-hidden="true">
                <span class="side-logo">02</span>
                <span class="side-item is-active"></span>
                <span class="side-item"></span>
                <span class="side-item"></span>
                <span class="side-item side-item-short"></span>
              </aside>
              <div class="system-content">
                <div class="system-heading">
                  <div>
                    <span class="micro-label">TRANSFER CENTER</span>
                    <strong>文件空间</strong>
                  </div>
                  <span class="sync-status"><i></i>SYNCED</span>
                </div>

                <div class="upload-file">
                  <span class="file-icon" aria-hidden="true">
                    <svg viewBox="0 0 24 24">
                      <path
                        d="M6 3h8l4 4v14H6V3Z"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.4"
                      />
                      <path d="M14 3v5h5" fill="none" stroke="currentColor" stroke-width="1.4" />
                    </svg>
                  </span>
                  <div class="file-copy">
                    <strong>product-notes.pdf</strong>
                    <span>18.4 MB · resumable upload</span>
                  </div>
                  <strong class="file-progress">100%</strong>
                </div>
                <div class="progress-track" aria-hidden="true"><span></span></div>

                <div class="share-link-card">
                  <div>
                    <span class="micro-label">SHARE LINK / 7 DAYS</span>
                    <strong>/shares/7K3P...Q9</strong>
                  </div>
                  <span class="copy-chip">COPY</span>
                </div>

                <div class="system-stats">
                  <div v-for="item in systemStats" :key="item.label">
                    <span>{{ item.label }}</span>
                    <strong>{{ item.value }}</strong>
                  </div>
                </div>
              </div>
            </div>
            <div class="system-footer">
              <span>INTERFACE STUDY / 2026</span>
              <span>020200 OS</span>
            </div>
          </div>
        </div>
      </section>

      <section id="about" class="section manifesto" aria-labelledby="about-title">
        <div class="section-head">
          <div>
            <p class="section-kicker reveal">03 — HOW I BUILD</p>
            <h2 class="reveal" id="about-title">好的系统不需要<br />被注意。</h2>
          </div>
          <blockquote class="reveal">
            它只是在该工作的时候，安静地工作；在该清晰的时候，毫不含糊。
          </blockquote>
        </div>

        <div class="principle-grid">
          <article v-for="item in principles" :key="item.index" class="principle reveal">
            <span>{{ item.index }}</span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
          </article>
        </div>
      </section>

      <section id="contact" class="contact-wrap" aria-labelledby="contact-title">
        <div class="contact">
          <div>
            <p class="section-kicker reveal">04 — START A CONVERSATION</p>
            <h2 class="reveal" id="contact-title">有合适的问题，<br />一起把它做出来。</h2>
            <p class="contact-intro reveal">
              无论是一个产品想法、一套后端系统，还是想把旧工具重新做好，都欢迎聊聊。
            </p>
          </div>
          <a
            class="contact-orb reveal"
            href="mailto:hello@020200.xyz"
            aria-label="发送邮件到 hello@020200.xyz"
          >
            <span>SAY<br />HELLO</span>
            <svg viewBox="0 0 64 64" aria-hidden="true">
              <path
                d="M17 32h30M35 20l12 12-12 12"
                fill="none"
                stroke="currentColor"
                stroke-width="2.4"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </a>
        </div>
      </section>
    </main>

    <footer class="site-footer">
      <div class="footer-brand">
        <span class="brand-mark" aria-hidden="true">02</span>
        <div>
          <strong>020200</strong>
          <span>把复杂留给系统，把简单留给使用。</span>
        </div>
      </div>
      <div class="footer-meta">
        <span>© {{ currentYear }} 020200</span>
        <span>BUILT WITH GO, GIN &amp; PATIENCE</span>
        <a href="#main">BACK TO TOP ↑</a>
      </div>
    </footer>
  </div>
</template>
