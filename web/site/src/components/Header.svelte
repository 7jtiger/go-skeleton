<script lang="ts">
  import { link, location } from 'svelte-spa-router'
  import Logo from './Logo.svelte'

  const navItems = [
    { href: '/', label: '홈' },
    { href: '/solutions', label: '솔루션' },
    { href: '/chat', label: '채팅 서비스' },
    { href: '/about', label: '회사 소개' },
    { href: '/contact', label: '문의' },
  ]

  let menuOpen = $state(false)

  function isActive(href: string): boolean {
    if (href === '/') return $location === '/' || $location === ''
    return $location.startsWith(href)
  }

  function closeMenu() {
    menuOpen = false
  }
</script>

<header class="header">
  <div class="container header-inner">
    <a href="/" use:link class="logo-link" onclick={closeMenu}>
      <Logo variant="light" size="md" />
    </a>

    <button class="menu-btn" aria-label="메뉴" onclick={() => (menuOpen = !menuOpen)}>
      <span class:open={menuOpen}></span>
    </button>

    <nav class:open={menuOpen}>
      {#each navItems as item}
        <a
          href={item.href}
          use:link
          class:active={isActive(item.href)}
          onclick={closeMenu}
        >
          {item.label}
        </a>
      {/each}
    </nav>
  </div>
</header>

<style>
  .header {
    position: sticky;
    top: 0;
    z-index: 100;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(12px);
    border-bottom: 1px solid var(--color-border);
    height: var(--header-height);
  }

  .header-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
  }

  .logo-link {
    display: flex;
    align-items: center;
  }

  nav {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  nav a {
    padding: 0.5rem 0.9rem;
    border-radius: 999px;
    font-size: 0.92rem;
    font-weight: 500;
    color: var(--color-text-muted);
    transition: color 0.2s, background 0.2s;
  }

  nav a:hover,
  nav a.active {
    color: var(--color-primary);
    background: var(--color-surface);
  }

  .menu-btn {
    display: none;
    width: 40px;
    height: 40px;
    border: none;
    background: transparent;
    cursor: pointer;
    position: relative;
  }

  .menu-btn span,
  .menu-btn span::before,
  .menu-btn span::after {
    display: block;
    width: 22px;
    height: 2px;
    background: var(--color-primary);
    border-radius: 2px;
    position: absolute;
    left: 9px;
    transition: transform 0.2s;
  }

  .menu-btn span {
    top: 19px;
  }

  .menu-btn span::before,
  .menu-btn span::after {
    content: '';
    left: 0;
  }

  .menu-btn span::before {
    top: -7px;
  }

  .menu-btn span::after {
    top: 7px;
  }

  .menu-btn span.open {
    background: transparent;
  }

  .menu-btn span.open::before {
    transform: rotate(45deg) translate(5px, 5px);
  }

  .menu-btn span.open::after {
    transform: rotate(-45deg) translate(5px, -5px);
  }

  @media (max-width: 768px) {
    .menu-btn {
      display: block;
    }

    nav {
      display: none;
      position: absolute;
      top: var(--header-height);
      left: 0;
      right: 0;
      flex-direction: column;
      background: #fff;
      border-bottom: 1px solid var(--color-border);
      padding: 1rem;
      gap: 0.25rem;
    }

    nav.open {
      display: flex;
    }

    nav a {
      width: 100%;
      text-align: center;
    }

  }
</style>
