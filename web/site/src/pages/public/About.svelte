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
      <h1 class="section-title">회사 소개</h1>
      <p class="section-subtitle">기술력과 신뢰를 바탕으로 성장하는 Livein</p>
    </div>
  </section>

  {#if loading}
    <div class="container loading">로딩 중...</div>
  {:else if content}
    <section class="section">
      <div class="container about-grid">
        <FadeIn>
          <div class="card">
            <h2>소개</h2>
            <p>{content.about.description}</p>
          </div>
        </FadeIn>
        <FadeIn>
          <div class="card mission">
            <h2>미션</h2>
            <p>{content.about.mission}</p>
          </div>
        </FadeIn>
        <FadeIn>
          <div class="card vision">
            <h2>비전</h2>
            <p>{content.about.vision}</p>
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

  .about-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.25rem;
  }

  h2 {
    margin: 0 0 0.75rem;
    color: var(--color-primary);
    font-size: 1.15rem;
  }

  p {
    margin: 0;
    color: var(--color-text-muted);
    line-height: 1.8;
  }

  .mission {
    border-left: 4px solid var(--color-accent);
  }

  .vision {
    border-left: 4px solid var(--color-success);
  }

  .loading {
    padding: 4rem 0;
    text-align: center;
    color: var(--color-text-muted);
  }
</style>
