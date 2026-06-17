<script lang="ts">
  import { link } from 'svelte-spa-router'
  import Logo from './Logo.svelte'
  import type { ContactSection } from '../lib/types'

  let { contact }: { contact?: ContactSection } = $props()
</script>

<footer class="footer">
  <div class="container footer-inner">
    <div class="brand">
      <Logo variant="dark" size="md" />
      <p class="tagline">웹 솔루션 · 실시간 채팅 서비스</p>
    </div>

    <div class="links">
      <a href="/solutions" use:link>솔루션</a>
      <a href="/chat" use:link>채팅 서비스</a>
      <a href="/about" use:link>회사 소개</a>
      <a href="/contact" use:link>문의</a>
    </div>

    <div class="contact">
      {#if contact?.email}
        <a href="mailto:{contact.email}">{contact.email}</a>
      {/if}
      {#if contact?.address}
        <span>{contact.address}</span>
      {/if}
      <a href="/admin/login" use:link class="admin-link">Admin</a>
    </div>
  </div>

  <div class="container copyright">
    <span>© {new Date().getFullYear()} LIVEIN. All rights reserved.</span>
  </div>
</footer>

<style>
  .footer {
    background: var(--color-primary);
    color: #cbd5e1;
    padding: 3rem 0 1.5rem;
    margin-top: auto;
  }

  .footer-inner {
    display: grid;
    grid-template-columns: 1.5fr 1fr 1fr;
    gap: 2rem;
    padding-bottom: 2rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .brand {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    align-items: flex-start;
  }

  .tagline {
    margin: 0;
    font-size: 0.9rem;
    color: #94a3b8;
  }

  .links {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .links a:hover {
    color: #fff;
  }

  .contact {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
  }

  .contact a:hover {
    color: #fff;
  }

  .admin-link {
    margin-top: 0.15rem;
    color: #93c5fd;
    font-weight: 600;
  }

  .copyright {
    padding-top: 1.25rem;
    font-size: 0.85rem;
    color: #94a3b8;
  }

  @media (max-width: 768px) {
    .footer-inner {
      grid-template-columns: 1fr;
    }
  }
</style>
