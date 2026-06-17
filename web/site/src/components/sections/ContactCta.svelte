<script lang="ts">
  import { link } from 'svelte-spa-router'
  import FadeIn from '../FadeIn.svelte'
</script>

<section class="section cta">
  <div class="container">
    <FadeIn>
      <div class="cta-box">
        <h2>프로젝트를 시작할 준비가 되셨나요?</h2>
        <p>웹 솔루션, 채팅 서비스, 맞춤 개발에 대해 상담해 드립니다.</p>
        <a href="/contact" use:link class="btn btn-primary">문의하기</a>
      </div>
    </FadeIn>
  </div>
</section>

<style>
  .cta {
    padding-bottom: 2rem;
  }

  .cta-box {
    text-align: center;
    padding: 3.5rem 2rem;
    border-radius: var(--radius-lg);
    background: linear-gradient(135deg, #eff6ff 0%, #f8fafc 100%);
    border: 1px solid var(--color-border);
  }

  h2 {
    margin: 0 0 0.75rem;
    color: var(--color-primary);
    font-size: clamp(1.5rem, 3vw, 2rem);
  }

  p {
    margin: 0 0 1.5rem;
    color: var(--color-text-muted);
  }
</style>
