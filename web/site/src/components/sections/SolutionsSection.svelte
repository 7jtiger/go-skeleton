<script lang="ts">
  import FadeIn from '../FadeIn.svelte'
  import type { Solution } from '../../lib/types'

  let { solutions }: { solutions: Solution[] } = $props()

  function statusLabel(status: string): string {
    return status === 'active' ? '운영 중' : '개발 중'
  }
</script>

<section class="section" id="solutions">
  <div class="container">
    <FadeIn>
      <h2 class="section-title">솔루션</h2>
      <p class="section-subtitle">지속적으로 개발·고도화하는 제품과 플랫폼</p>
    </FadeIn>

    <div class="grid">
      {#each solutions as solution}
        <FadeIn>
          <article class="card solution-card">
            <div class="top">
              <h3>{solution.name}</h3>
              <span class="status" class:active={solution.status === 'active'}>
                {statusLabel(solution.status)}
              </span>
            </div>
            <p>{solution.summary}</p>
          </article>
        </FadeIn>
      {/each}
    </div>
  </div>
</section>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.25rem;
  }

  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.75rem;
  }

  h3 {
    margin: 0;
    color: var(--color-primary);
  }

  .status {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.25rem 0.65rem;
    border-radius: 999px;
    background: #fef3c7;
    color: #b45309;
    white-space: nowrap;
  }

  .status.active {
    background: #d1fae5;
    color: #047857;
  }

  .solution-card p {
    margin: 0;
    color: var(--color-text-muted);
    font-size: 0.95rem;
  }
</style>
