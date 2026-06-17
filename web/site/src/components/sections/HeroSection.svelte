<script lang="ts">
  import { link } from 'svelte-spa-router'
  import type { HeroSection } from '../lib/types'

  let { hero }: { hero: HeroSection } = $props()
</script>

<section class="hero">
  <div class="container hero-inner">
    <div class="hero-content">
      <span class="badge">Web Solutions & Real-time Chat</span>
      <h1>{hero.title}</h1>
      <p>{hero.subtitle}</p>
      <div class="actions">
        <a href="/solutions" use:link class="btn btn-primary">{hero.ctaPrimary}</a>
        <a href="/contact" use:link class="btn btn-outline">{hero.ctaSecondary}</a>
      </div>
    </div>
    <div class="hero-visual" aria-hidden="true">
      <div class="grid-card c1"></div>
      <div class="grid-card c2"></div>
      <div class="grid-card c3"></div>
      <div class="grid-card c4"></div>
    </div>
  </div>
</section>

<style>
  .hero {
    padding: 5rem 0 4rem;
    background: linear-gradient(180deg, #f8fafc 0%, #fff 100%);
  }

  .hero-inner {
    display: grid;
    grid-template-columns: 1.1fr 0.9fr;
    gap: 3rem;
    align-items: center;
  }

  .badge {
    display: inline-block;
    padding: 0.35rem 0.85rem;
    border-radius: 999px;
    background: rgba(37, 99, 235, 0.1);
    color: var(--color-accent);
    font-size: 0.82rem;
    font-weight: 600;
    margin-bottom: 1.25rem;
  }

  h1 {
    font-size: clamp(2rem, 4.5vw, 3rem);
    line-height: 1.15;
    letter-spacing: -0.03em;
    color: var(--color-primary);
    margin: 0 0 1rem;
  }

  p {
    color: var(--color-text-muted);
    font-size: 1.1rem;
    margin: 0 0 2rem;
    max-width: 520px;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .hero-visual {
    position: relative;
    height: 320px;
  }

  .grid-card {
    position: absolute;
    border-radius: var(--radius-lg);
    border: 1px solid var(--color-border);
    background: #fff;
    box-shadow: var(--shadow-md);
  }

  .c1 {
    width: 55%;
    height: 45%;
    top: 10%;
    left: 5%;
    background: linear-gradient(135deg, #eff6ff, #fff);
  }

  .c2 {
    width: 50%;
    height: 40%;
    top: 35%;
    right: 0;
    background: linear-gradient(135deg, #f0fdf4, #fff);
  }

  .c3 {
    width: 42%;
    height: 35%;
    bottom: 5%;
    left: 15%;
    background: linear-gradient(135deg, #f8fafc, #fff);
  }

  .c4 {
    width: 28%;
    height: 28%;
    top: 0;
    right: 20%;
    border-color: var(--color-accent);
    opacity: 0.5;
  }

  @media (max-width: 900px) {
    .hero-inner {
      grid-template-columns: 1fr;
    }

    .hero-visual {
      height: 220px;
    }
  }
</style>
