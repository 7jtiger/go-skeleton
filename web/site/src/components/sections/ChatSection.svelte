<script lang="ts">
  import FadeIn from '../FadeIn.svelte'
  import type { ChatService } from '../../lib/types'

  let { chatService }: { chatService: ChatService } = $props()
</script>

<section class="section chat-section" id="chat">
  <div class="container chat-inner">
    <FadeIn>
      <div class="content">
        <h2 class="section-title">{chatService.title}</h2>
        <p class="section-subtitle">{chatService.description}</p>

        <ul class="features">
          {#each chatService.features as feature}
            <li>{feature}</li>
          {/each}
        </ul>

        <div class="stack">
          {#each chatService.techStack as tech}
            <span class="tag">{tech}</span>
          {/each}
        </div>
      </div>
    </FadeIn>

    <FadeIn>
      <div class="diagram card" aria-hidden="true">
        <div class="node client">Client</div>
        <div class="arrow">↔</div>
        <div class="node gateway">Gateway</div>
        <div class="arrow">↔</div>
        <div class="nodes">
          <div class="node">Redis</div>
          <div class="node">MySQL</div>
        </div>
      </div>
    </FadeIn>
  </div>
</section>

<style>
  .chat-section {
    background: var(--color-primary);
    color: #e2e8f0;
  }

  .chat-inner {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 3rem;
    align-items: center;
  }

  .section-title {
    color: #fff;
  }

  .section-subtitle {
    color: #94a3b8;
  }

  .features {
    margin: 0 0 1.5rem;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 0.6rem;
  }

  .features li::before {
    content: '✓ ';
    color: var(--color-success);
    font-weight: 700;
  }

  .stack {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .tag {
    padding: 0.35rem 0.75rem;
    border-radius: 999px;
    background: rgba(37, 99, 235, 0.25);
    color: #93c5fd;
    font-size: 0.85rem;
    font-weight: 500;
  }

  .diagram {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
    padding: 2rem;
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.1);
  }

  .node {
    padding: 0.6rem 1.25rem;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    font-size: 0.9rem;
    font-weight: 600;
  }

  .gateway {
    background: rgba(37, 99, 235, 0.3);
    border-color: #3b82f6;
    color: #bfdbfe;
  }

  .nodes {
    display: flex;
    gap: 0.75rem;
  }

  .arrow {
    color: #64748b;
    font-size: 1.25rem;
  }

  @media (max-width: 900px) {
    .chat-inner {
      grid-template-columns: 1fr;
    }
  }
</style>
