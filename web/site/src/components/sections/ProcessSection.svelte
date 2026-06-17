<script lang="ts">
  import FadeIn from '../FadeIn.svelte'
  import type { ProcessStep } from '../../lib/types'

  let { process }: { process: ProcessStep[] } = $props()
</script>

<section class="section" id="process">
  <div class="container">
    <FadeIn>
      <h2 class="section-title">개발 프로세스</h2>
      <p class="section-subtitle">체계적인 프로세스로 프로젝트를 성공적으로 완수합니다.</p>
    </FadeIn>

    <div class="steps">
      {#each process as step}
        <FadeIn>
          <article class="step card">
            <span class="num">{step.step}</span>
            <h3>{step.title}</h3>
            <p>{step.description}</p>
          </article>
        </FadeIn>
      {/each}
    </div>
  </div>
</section>

<style>
  .steps {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .step {
    position: relative;
    padding-top: 2.5rem;
  }

  .num {
    position: absolute;
    top: 1rem;
    left: 1.5rem;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--color-accent);
    color: #fff;
    display: grid;
    place-items: center;
    font-size: 0.85rem;
    font-weight: 700;
  }

  h3 {
    margin: 0 0 0.5rem;
    color: var(--color-primary);
    font-size: 1.05rem;
  }

  p {
    margin: 0;
    color: var(--color-text-muted);
    font-size: 0.92rem;
  }
</style>
