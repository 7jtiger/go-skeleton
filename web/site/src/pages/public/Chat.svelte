<script lang="ts">
  import { onMount } from 'svelte'
  import PublicLayout from '../../components/PublicLayout.svelte'
  import FadeIn from '../../components/FadeIn.svelte'
  import { fetchContent } from '../../lib/api'
  import type { Content } from '../../lib/types'

  let content = $state<Content | null>(null)
  let loading = $state(true)

  onMount(async () => {
    content = await fetchContent()
    loading = false
  })
</script>

<PublicLayout {content}>
  <section class="section page-hero">
    <div class="container">
      <h1 class="section-title">채팅 서비스</h1>
      <p class="section-subtitle">실시간 커뮤니케이션을 위한 엔터프라이즈급 아키텍처</p>
    </div>
  </section>

  {#if loading}
    <div class="container loading">로딩 중...</div>
  {:else if content}
    <section class="section">
      <div class="container layout">
        <FadeIn>
          <div class="info card">
            <h2>{content.chatService.title}</h2>
            <p>{content.chatService.description}</p>

            <h3>주요 기능</h3>
            <ul>
              {#each content.chatService.features as feature}
                <li>{feature}</li>
              {/each}
            </ul>
          </div>
        </FadeIn>

        <FadeIn>
          <div class="arch card">
            <h3>아키텍처</h3>
            <div class="flow">
              <div class="box">Client App</div>
              <div class="line"></div>
              <div class="box highlight">API Gateway</div>
              <div class="line"></div>
              <div class="row">
                <div class="box">WebSocket</div>
                <div class="box">WebRTC</div>
              </div>
              <div class="line"></div>
              <div class="row">
                <div class="box">Redis</div>
                <div class="box">MySQL</div>
              </div>
            </div>

            <h3>기술 스택</h3>
            <div class="tags">
              {#each content.chatService.techStack as tech}
                <span class="tag">{tech}</span>
              {/each}
            </div>
          </div>
        </FadeIn>
      </div>
    </section>
  {/if}
</PublicLayout>

<style>
  .page-hero {
    padding-top: 3rem;
    padding-bottom: 1rem;
    background: var(--color-surface);
  }

  .layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
    align-items: start;
  }

  h2 {
    margin: 0 0 1rem;
    color: var(--color-primary);
  }

  h3 {
    margin: 1.5rem 0 0.75rem;
    font-size: 1rem;
    color: var(--color-primary);
  }

  .info p {
    color: var(--color-text-muted);
    margin: 0;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 0.5rem;
  }

  ul li::before {
    content: '✓ ';
    color: var(--color-success);
    font-weight: 700;
  }

  .flow {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
  }

  .box {
    padding: 0.6rem 1.25rem;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    font-size: 0.88rem;
    font-weight: 600;
    text-align: center;
    min-width: 120px;
  }

  .highlight {
    background: rgba(37, 99, 235, 0.1);
    border-color: var(--color-accent);
    color: var(--color-accent);
  }

  .row {
    display: flex;
    gap: 0.75rem;
  }

  .line {
    width: 2px;
    height: 20px;
    background: var(--color-border);
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .tag {
    padding: 0.35rem 0.75rem;
    border-radius: 999px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    font-size: 0.85rem;
  }

  .loading {
    padding: 4rem 0;
    text-align: center;
    color: var(--color-text-muted);
  }

  @media (max-width: 900px) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
</style>
