<script lang="ts">
  import { onMount } from 'svelte'
  import PublicLayout from '../../components/PublicLayout.svelte'
  import { fetchContent, submitContact } from '../../lib/api'
  import type { Content } from '../../lib/types'

  let content = $state<Content | null>(null)
  let name = $state('')
  let email = $state('')
  let message = $state('')
  let loading = $state(false)
  let success = $state(false)
  let error = $state('')

  onMount(async () => {
    content = await fetchContent()
  })

  async function handleSubmit(e: Event) {
    e.preventDefault()
    loading = true
    error = ''
    success = false
    try {
      await submitContact({ name, email, message })
      success = true
      name = ''
      email = ''
      message = ''
    } catch (err) {
      error = err instanceof Error ? err.message : '문의 접수에 실패했습니다'
    } finally {
      loading = false
    }
  }
</script>

<PublicLayout {content}>
  <section class="section page-hero">
    <div class="container">
      <h1 class="section-title">문의하기</h1>
      <p class="section-subtitle">프로젝트나 서비스에 대해 편하게 문의해 주세요.</p>
    </div>
  </section>

  <section class="section">
    <div class="container layout">
      <div class="info card">
        <h2>연락처</h2>
        {#if content?.contact.email}
          <p>
            <strong>이메일</strong><br />
            <a href="mailto:{content.contact.email}">{content.contact.email}</a>
          </p>
        {/if}
        {#if content?.contact.address}
          <p>
            <strong>주소</strong><br />
            {content.contact.address}
          </p>
        {/if}
      </div>

      <form class="card form" onsubmit={handleSubmit}>
        <div class="form-group">
          <label for="name">이름</label>
          <input id="name" type="text" bind:value={name} required />
        </div>
        <div class="form-group">
          <label for="email">이메일</label>
          <input id="email" type="email" bind:value={email} required />
        </div>
        <div class="form-group">
          <label for="message">문의 내용</label>
          <textarea id="message" bind:value={message} required></textarea>
        </div>

        {#if success}
          <p class="msg success">문의가 접수되었습니다. 빠른 시일 내에 답변드리겠습니다.</p>
        {/if}
        {#if error}
          <p class="msg error">{error}</p>
        {/if}

        <button type="submit" class="btn btn-primary" disabled={loading}>
          {loading ? '전송 중...' : '문의 보내기'}
        </button>
      </form>
    </div>
  </section>
</PublicLayout>

<style>
  .page-hero {
    padding-top: 3rem;
    padding-bottom: 1rem;
    background: var(--color-surface);
  }

  .layout {
    display: grid;
    grid-template-columns: 1fr 1.5fr;
    gap: 1.5rem;
    align-items: start;
  }

  .info h2 {
    margin: 0 0 1.25rem;
    color: var(--color-primary);
  }

  .info p {
    margin: 0 0 1rem;
    color: var(--color-text-muted);
    line-height: 1.7;
  }

  .info a {
    color: var(--color-accent);
  }

  .form .btn {
    width: 100%;
  }

  .msg {
    margin: 0 0 1rem;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    font-size: 0.9rem;
  }

  .msg.success {
    background: #d1fae5;
    color: #047857;
  }

  .msg.error {
    background: #fee2e2;
    color: #b91c1c;
  }

  @media (max-width: 768px) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
</style>
