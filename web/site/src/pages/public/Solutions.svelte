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

  function statusLabel(status: string): string {
    return status === 'active' ? '운영 중' : '개발 중'
  }
</script>

<PublicLayout {content}>
  <section class="section page-hero">
    <div class="container">
      <h1 class="section-title">솔루션</h1>
      <p class="section-subtitle">Livein이 개발·운영하는 솔루션과 플랫폼을 소개합니다.</p>
    </div>
  </section>

  {#if loading}
    <div class="container loading">로딩 중...</div>
  {:else if content}
    <section class="section">
      <div class="container grid">
        {#each content.solutions as solution}
          <FadeIn>
            <article class="card detail-card">
              <div class="top">
                <h2>{solution.name}</h2>
                <span class="status" class:active={solution.status === 'active'}>
                  {statusLabel(solution.status)}
                </span>
              </div>
              <p>{solution.summary}</p>
              <span class="id">ID: {solution.id}</span>
            </article>
          </FadeIn>
        {/each}
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

  .grid {
    display: grid;
    gap: 1.25rem;
  }

  .detail-card h2 {
    margin: 0;
    font-size: 1.35rem;
    color: var(--color-primary);
  }

  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .status {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.25rem 0.65rem;
    border-radius: 999px;
    background: #fef3c7;
    color: #b45309;
  }

  .status.active {
    background: #d1fae5;
    color: #047857;
  }

  .detail-card p {
    margin: 0 0 1rem;
    color: var(--color-text-muted);
    line-height: 1.7;
  }

  .id {
    font-size: 0.8rem;
    color: #94a3b8;
    font-family: monospace;
  }

  .loading {
    padding: 4rem 0;
    text-align: center;
    color: var(--color-text-muted);
  }
</style>
