<script lang="ts">
  import FadeIn from '../FadeIn.svelte'
  import type { Service } from '../../lib/types'

  let { services }: { services: Service[] } = $props()

  const icons: Record<string, string> = {
    code: '</>',
    message: '💬',
    puzzle: '🧩',
    shield: '🛡',
  }
</script>

<section class="section services" id="services">
  <div class="container">
    <FadeIn>
      <h2 class="section-title">서비스</h2>
      <p class="section-subtitle">기획부터 운영까지, 안정적인 웹 솔루션을 제공합니다.</p>
    </FadeIn>

    <div class="grid">
      {#each services as service}
        <FadeIn>
          <article class="card service-card">
            <span class="icon">{icons[service.icon] ?? '◆'}</span>
            <h3>{service.title}</h3>
            <p>{service.description}</p>
          </article>
        </FadeIn>
      {/each}
    </div>
  </div>
</section>

<style>
  .services {
    background: var(--color-surface);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 1.25rem;
  }

  .service-card h3 {
    margin: 0.75rem 0 0.5rem;
    font-size: 1.1rem;
    color: var(--color-primary);
  }

  .service-card p {
    margin: 0;
    color: var(--color-text-muted);
    font-size: 0.95rem;
  }

  .icon {
    font-size: 1.5rem;
    display: inline-block;
  }
</style>
