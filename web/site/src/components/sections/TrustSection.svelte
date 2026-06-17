<script lang="ts">
  import FadeIn from '../FadeIn.svelte'
  import type { TrustSection } from '../../lib/types'

  let { trust }: { trust: TrustSection } = $props()
</script>

<section class="section trust" id="trust">
  <div class="container">
    <FadeIn>
      <h2 class="section-title">신뢰할 수 있는 기술 파트너</h2>
      <p class="section-subtitle">보안과 안정성을 최우선으로 설계합니다.</p>
    </FadeIn>

    <div class="metrics">
      {#each trust.metrics as metric}
        <FadeIn>
          <div class="metric card">
            <span class="value">{metric.value}</span>
            <span class="label">{metric.label}</span>
          </div>
        </FadeIn>
      {/each}
    </div>

    <div class="badges">
      {#each trust.badges as badge}
        <span class="badge">{badge}</span>
      {/each}
    </div>
  </div>
</section>

<style>
  .trust {
    background: var(--color-surface);
  }

  .metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }

  .metric {
    text-align: center;
    padding: 2rem 1rem;
  }

  .value {
    display: block;
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--color-accent);
    margin-bottom: 0.35rem;
  }

  .label {
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }

  .badges {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    justify-content: center;
  }

  .badge {
    padding: 0.5rem 1rem;
    border-radius: 999px;
    border: 1px solid var(--color-border);
    background: #fff;
    font-size: 0.88rem;
    font-weight: 500;
    color: var(--color-primary);
  }
</style>
