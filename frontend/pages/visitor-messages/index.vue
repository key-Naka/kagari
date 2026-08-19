<script setup lang="ts">
import { reactive, shallowRef } from 'vue'

interface PublicVisitorMessage {
  id: number
  nickname: string
  content: string
  createdAt: string
}

interface VisitorMessageForm {
  nickname: string
  email: string
  content: string
}

interface ApiFailure {
  data?: { error?: string }
  statusCode?: number
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const form = reactive<VisitorMessageForm>({ nickname: '', email: '', content: '' })
const isSubmitting = shallowRef(false)
const submitError = shallowRef('')
const submitSuccess = shallowRef('')

const { data: messages, status, error, refresh } = await useFetch<PublicVisitorMessage[]>(`${apiBase}/api/v1/visitor-messages`, {
  key: 'visitor-messages',
  default: () => [],
})

usePublicSeo({
  title: '访客留言 · Kagari',
  description: '向 Kagari 的公开档案投递一条 Visitor Message；昵称与邮箱均可选，邮箱永不公开。',
})

function displayName(message: PublicVisitorMessage): string {
  return message.nickname || '匿名访客'
}

function displayDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Shanghai',
  }).format(new Date(value))
}

function submissionError(error: unknown): string {
  const failure = error as ApiFailure
  if (failure?.statusCode === 429) return '发送过于频繁，请稍后再试。'
  if (failure?.statusCode === 503) return '留言服务暂时不可用，请稍后再试。'
  return failure?.data?.error || '讯号发送失败，请检查网络后重试。'
}

async function refreshMessages(): Promise<void> {
  await refresh()
}

async function submitVisitorMessage(): Promise<void> {
  submitError.value = ''
  submitSuccess.value = ''
  isSubmitting.value = true
  try {
    const created = await $fetch<PublicVisitorMessage>(`${apiBase}/api/v1/visitor-messages`, {
      method: 'POST',
      body: {
        nickname: form.nickname,
        email: form.email,
        content: form.content,
      },
    })
    form.nickname = ''
    form.email = ''
    form.content = ''
    messages.value = [created, ...messages.value]
    submitSuccess.value = '讯号已进入公开档案。'
  }
  catch (error) {
    submitError.value = submissionError(error)
  }
  finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <main class="message-page">
    <div class="message-page__grain" aria-hidden="true" />
    <div class="message-page__shell">
      <header class="message-page__header">
        <div>
          <p class="message-page__index">ARCHIVE / 08 · OPEN FREQUENCY</p>
          <h1>访客留言</h1>
          <p class="message-page__intro">这里收录从站外抵达的公开讯号。你可以保持匿名，也可以留下一个昵称；每一条讯息都会立即进入这份档案。</p>
        </div>
        <div class="message-page__counter" aria-label="公开留言数量">
          <span>{{ messages.length.toString().padStart(2, '0') }}</span>
          <small>PUBLIC SIGNALS</small>
        </div>
      </header>

      <section class="transmission" aria-labelledby="transmission-heading">
        <div class="transmission__form-panel">
          <div class="section-heading">
            <p>TRANSMISSION FORM</p>
            <h2 id="transmission-heading">留下一个讯号</h2>
          </div>

          <p v-if="submitError" class="notice notice--error" role="alert">{{ submitError }}</p>
          <p v-if="submitSuccess" class="notice notice--success" role="status">{{ submitSuccess }}</p>

          <form class="message-form" @submit.prevent="submitVisitorMessage">
            <div class="message-form__pair">
              <label>
                <span>昵称 <i>可选</i></span>
                <input v-model.trim="form.nickname" name="nickname" autocomplete="nickname" maxlength="80" placeholder="匿名访客" :disabled="isSubmitting">
              </label>
              <label>
                <span>邮箱 <i>私有 / 可选</i></span>
                <input v-model.trim="form.email" name="email" type="email" autocomplete="email" maxlength="254" placeholder="name@example.com" :disabled="isSubmitting" aria-describedby="email-privacy">
              </label>
            </div>
            <label>
              <span>公开讯息 <i>{{ form.content.length }} / 1000</i></span>
              <textarea v-model="form.content" name="content" required maxlength="1000" rows="7" placeholder="写下一些你愿意公开留下的内容……" :disabled="isSubmitting" />
            </label>
            <div class="message-form__actions">
              <p id="email-privacy">邮箱不会公开，仅供站点所有者在必要时联系你。</p>
              <button type="submit" class="cursor-target" :disabled="isSubmitting">
                <span>{{ isSubmitting ? '传输中' : '发送讯号' }}</span>
                <b aria-hidden="true">↗</b>
              </button>
            </div>
          </form>
        </div>

        <aside class="privacy-card" aria-labelledby="privacy-heading">
          <p class="privacy-card__eyebrow">BOUNDARY / 03</p>
          <h2 id="privacy-heading">数据边界</h2>
          <dl>
            <div><dt>昵称</dt><dd>可留空；空白时以“匿名访客”展示。</dd></div>
            <div><dt>邮箱</dt><dd>可选并仅对管理员可见，永不进入公开 API。</dd></div>
            <div><dt>Rate Limit</dt><dd>Redis 会按 IP 与当前提交路由短时计数，降低滥用。</dd></div>
          </dl>
          <p>所有输入都会经过长度校验和 HTML 清理。本站不使用 CAPTCHA，也不设置审核队列。</p>
        </aside>
      </section>

      <section class="signal-archive" aria-labelledby="signal-heading">
        <div class="section-heading section-heading--row">
          <div><p>RECEIVED TRANSMISSIONS</p><h2 id="signal-heading">公开讯号</h2></div>
          <button type="button" class="archive-refresh cursor-target" :disabled="status === 'pending'" @click="refreshMessages">{{ status === 'pending' ? '同步中' : '重新同步' }}</button>
        </div>

        <p v-if="error" class="notice notice--error" role="alert">公开讯号暂时无法读取。你仍然可以稍后重试。</p>
        <p v-else-if="status === 'pending' && messages.length === 0" class="signal-empty" role="status">正在调谐公开频率……</p>
        <p v-else-if="messages.length === 0" class="signal-empty">这里尚无讯号。你可以成为第一位访客。</p>
        <ol v-else class="signal-list">
          <li v-for="(message, index) in messages" :key="message.id" class="signal-entry">
            <div class="signal-entry__meta">
              <span>{{ String(messages.length - index).padStart(3, '0') }}</span>
              <p>{{ displayName(message) }}</p>
              <time :datetime="message.createdAt">{{ displayDate(message.createdAt) }}</time>
            </div>
            <p class="signal-entry__content">{{ message.content }}</p>
          </li>
        </ol>
      </section>
    </div>
  </main>
</template>

<style scoped>
.message-page{position:relative;min-height:100vh;overflow:hidden;background:radial-gradient(circle at 82% 4%,rgba(124,58,237,.13),transparent 28rem),radial-gradient(circle at 8% 58%,rgba(217,119,6,.07),transparent 32rem),#09090b;color:#f4f4f5}.message-page__grain{position:absolute;inset:0;opacity:.17;background-image:repeating-linear-gradient(0deg,transparent,transparent 3px,rgba(255,255,255,.02) 4px);pointer-events:none}.message-page__shell{position:relative;width:min(82rem,calc(100% - 2rem));margin:0 auto;padding:3.5rem 0 9rem}.message-page__header{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:end;gap:4rem;padding:1rem 0 3rem;border-bottom:1px solid #2a2a30}.message-page__index,.section-heading>p,.privacy-card__eyebrow{color:#a78bfa;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.64rem;letter-spacing:.25em}.message-page__header h1{margin:.85rem 0 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.6rem,9vw,7.8rem);font-weight:400;letter-spacing:-.065em;line-height:.88}.message-page__intro{max-width:42rem;margin:1.5rem 0 0;color:#a1a1aa;font-size:.94rem;line-height:1.9}.message-page__counter{display:grid;justify-items:end}.message-page__counter span{font-family:Georgia,'Times New Roman',serif;font-size:5rem;line-height:.8}.message-page__counter small{margin-top:1rem;color:#71717a;font-family:ui-monospace,monospace;font-size:.65rem;letter-spacing:.18em}.transmission{display:grid;grid-template-columns:minmax(0,1.65fr) minmax(18rem,.75fr);gap:1px;margin-top:3.5rem;border:1px solid #2a2a30;background:#2a2a30}.transmission__form-panel,.privacy-card{background:rgba(9,9,11,.96)}.transmission__form-panel{padding:clamp(1.5rem,4vw,3rem)}.section-heading h2,.privacy-card h2{margin:.55rem 0 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.9rem,4vw,3rem);font-weight:400}.message-form{display:grid;gap:1.3rem;margin-top:2.5rem}.message-form__pair{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}.message-form label{display:grid;gap:.6rem}.message-form label>span{display:flex;justify-content:space-between;color:#d4d4d8;font-size:.76rem;letter-spacing:.08em}.message-form label i{color:#71717a;font-family:ui-monospace,monospace;font-size:.62rem;font-style:normal}.message-form input,.message-form textarea{width:100%;border:1px solid #3f3f46;border-radius:0;background:#0e0e11;padding:.9rem 1rem;color:#f4f4f5;outline:none;transition:border-color .18s,box-shadow .18s}.message-form textarea{resize:vertical;line-height:1.75}.message-form input:focus,.message-form textarea:focus{border-color:#a78bfa;box-shadow:0 0 0 3px rgba(167,139,250,.1)}.message-form__actions{display:flex;align-items:center;justify-content:space-between;gap:2rem;border-top:1px solid #27272a;padding-top:1.25rem}.message-form__actions p{max-width:26rem;margin:0;color:#71717a;font-size:.72rem;line-height:1.65}.message-form button{display:flex;min-width:9.5rem;align-items:center;justify-content:space-between;gap:1.5rem;border:1px solid #c4b5fd;background:#c4b5fd;padding:.8rem 1rem;color:#18181b;font-size:.75rem;font-weight:700;letter-spacing:.08em;transition:background .18s,transform .18s}.message-form button:hover{background:#ddd6fe;transform:translateY(-2px)}.message-form button:disabled,.archive-refresh:disabled{cursor:wait;opacity:.55}.privacy-card{padding:clamp(1.5rem,3vw,2.4rem)}.privacy-card dl{display:grid;margin:2rem 0 1.5rem}.privacy-card dl div{border-top:1px solid #2d2d32;padding:1rem 0}.privacy-card dt{color:#d4d4d8;font-family:ui-monospace,monospace;font-size:.68rem;letter-spacing:.12em}.privacy-card dd,.privacy-card>p:last-child{margin:.45rem 0 0;color:#85858d;font-size:.75rem;line-height:1.7}.signal-archive{margin-top:5rem}.section-heading--row{display:flex;align-items:end;justify-content:space-between;gap:2rem;border-bottom:1px solid #2a2a30;padding-bottom:1.25rem}.archive-refresh{border:0;background:transparent;color:#a78bfa;font-family:ui-monospace,monospace;font-size:.66rem;letter-spacing:.12em;cursor:pointer}.signal-list{margin:0;padding:0;list-style:none}.signal-entry{display:grid;grid-template-columns:minmax(13rem,.65fr) minmax(0,1.65fr);gap:2rem;border-bottom:1px solid #27272a;padding:2rem 0}.signal-entry__meta{display:grid;grid-template-columns:3rem 1fr;align-content:start;gap:.45rem 1rem}.signal-entry__meta span{grid-row:1/3;color:#6d5ca7;font-family:Georgia,serif;font-size:1.7rem}.signal-entry__meta p{margin:0;color:#d4d4d8;font-size:.82rem}.signal-entry__meta time{color:#62626a;font-family:ui-monospace,monospace;font-size:.62rem}.signal-entry__content{margin:0;color:#bcbcc2;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1rem,2vw,1.28rem);line-height:1.75;white-space:pre-wrap}.signal-empty{border-bottom:1px solid #27272a;margin:0;padding:3rem 0;color:#71717a}.notice{margin:1.25rem 0 0;border:1px solid;padding:.85rem 1rem;font-size:.78rem}.notice--error{border-color:rgba(251,113,133,.35);background:rgba(136,19,55,.14);color:#fecdd3}.notice--success{border-color:rgba(52,211,153,.3);background:rgba(6,78,59,.15);color:#a7f3d0}@media(max-width:760px){.message-page__shell{padding-top:2rem}.message-page__header{grid-template-columns:1fr;gap:2rem}.message-page__counter{display:none}.transmission{grid-template-columns:1fr}.message-form__pair{grid-template-columns:1fr}.message-form__actions{align-items:stretch;flex-direction:column}.message-form button{width:100%}.signal-entry{grid-template-columns:1fr;gap:1rem}.section-heading--row{align-items:start;flex-direction:column}}@media(prefers-reduced-motion:reduce){.message-form button,.message-form input,.message-form textarea{transition:none}}
</style>
