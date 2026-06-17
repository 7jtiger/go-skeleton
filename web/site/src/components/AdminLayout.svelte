<script lang="ts">
  import type { Snippet } from 'svelte'
  import { onMount } from 'svelte'
  import { link, location, pop, push } from 'svelte-spa-router'
  import Logo from './Logo.svelte'
  import { clearToken, isTokenValid } from '../lib/auth'

  let {
    title = '',
    showBack = false,
    requiresAuth = true,
    showLogout = true,
    children,
    actions,
  }: {
    title?: string
    showBack?: boolean
    requiresAuth?: boolean
    showLogout?: boolean
    children?: Snippet
    actions?: Snippet
  } = $props()

  const navItems = [
    { href: '/admin', label: '대시보드', exact: true },
    { href: '/admin/content', label: '콘텐츠 편집' },
    { href: '/admin/inquiries', label: '문의 목록' },
  ]

  onMount(() => {
    if (requiresAuth && !isTokenValid()) {
      push('/admin/login')
    }
  })

  function navigate(e: MouseEvent, href: string) {
    e.preventDefault()
    void push(href)
  }

  function goBack(e: MouseEvent) {
    e.preventDefault()
    // pop()은 실패하지 않고 no-op일 수 있어 history 길이를 먼저 확인한다.
    if (window.history.length > 1) {
      void pop()
      return
    }
    void push('/admin')
  }

  function goDashboard(e: MouseEvent) {
    e.preventDefault()
    void push('/admin')
  }

  function logout() {
    clearToken()
    void push('/admin/login')
  }

  function isActive(href: string, exact = false): boolean {
    const path = $location
    if (exact) return path === href
    return path === href || path.startsWith(`${href}/`)
  }
</script>

<div class="admin-layout">
  <header class="admin-header">
    <div class="container header-top">
      <a href="/admin" class="logo-link" onclick={goDashboard}>
        <Logo variant="light" size="sm" />
      </a>

      <div class="header-actions">
        <a href="/" use:link class="site-link">홈페이지</a>
        {#if showLogout}
          <button type="button" class="btn btn-outline" onclick={logout}>로그아웃</button>
        {/if}
      </div>
    </div>

    <div class="container nav-bar">
      <nav class="admin-nav" aria-label="관리자 메뉴">
        {#each navItems as item}
          <a
            href={item.href}
            class:active={isActive(item.href, item.exact)}
            onclick={(e) => navigate(e, item.href)}
          >
            {item.label}
          </a>
        {/each}
      </nav>
    </div>
  </header>

  <div class="container admin-content">
    {#if showBack || title || actions}
      <div class="page-toolbar">
        <div class="page-title">
          {#if showBack}
            <button type="button" class="back-btn" onclick={goBack}>← 뒤로</button>
          {/if}
          {#if title}
            <h1>{title}</h1>
          {/if}
        </div>
        {#if actions}
          <div class="page-actions">
            {@render actions()}
          </div>
        {/if}
      </div>
    {/if}

    {@render children?.()}
  </div>
</div>

<style>
  .admin-layout {
    min-height: 100vh;
    background: var(--color-surface);
  }

  .admin-header {
    background: #fff;
    border-bottom: 1px solid var(--color-border);
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .header-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.875rem 0;
    gap: 1rem;
  }

  .logo-link {
    display: flex;
    align-items: center;
    cursor: pointer;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .site-link {
    font-size: 0.88rem;
    color: var(--color-text-muted);
    padding: 0.4rem 0.75rem;
    border-radius: 999px;
    transition: color 0.2s, background 0.2s;
  }

  .site-link:hover {
    color: var(--color-primary);
    background: var(--color-surface);
  }

  .nav-bar {
    padding-bottom: 0;
    overflow-x: auto;
    background: #fff;
  }

  .admin-nav {
    display: flex;
    gap: 0.25rem;
    border-top: 1px solid var(--color-border);
    border-bottom: 1px solid var(--color-border);
    padding: 0.35rem 0;
  }

  .admin-nav a {
    padding: 0.6rem 1rem;
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--color-text-muted);
    border-radius: 8px 8px 0 0;
    white-space: nowrap;
    border-bottom: 2px solid transparent;
    transition: color 0.2s, border-color 0.2s, background 0.2s;
  }

  .admin-nav a:hover {
    color: var(--color-primary);
    background: var(--color-surface);
  }

  .admin-nav a.active {
    color: var(--color-accent);
    border-bottom-color: var(--color-accent);
    font-weight: 600;
  }

  .admin-content {
    padding: 1.5rem 1rem 4rem;
    max-width: 960px;
  }

  .page-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.5rem;
    flex-wrap: wrap;
  }

  .page-title {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .page-title h1 {
    margin: 0;
    font-size: 1.35rem;
    color: var(--color-primary);
  }

  .back-btn {
    border: 1px solid var(--color-border);
    background: #fff;
    color: var(--color-text-muted);
    padding: 0.45rem 0.85rem;
    border-radius: 8px;
    font-size: 0.88rem;
    cursor: pointer;
    transition: color 0.2s, border-color 0.2s;
  }

  .back-btn:hover {
    color: var(--color-primary);
    border-color: var(--color-accent);
  }

  .page-actions {
    display: flex;
    gap: 0.5rem;
  }

  @media (max-width: 640px) {
    .header-actions .btn {
      padding: 0.5rem 0.75rem;
      font-size: 0.85rem;
    }

    .site-link {
      display: none;
    }
  }
</style>
