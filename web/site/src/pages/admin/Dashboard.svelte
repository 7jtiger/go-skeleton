<script lang="ts">
  import AdminLayout from '../../components/AdminLayout.svelte'
  import { push } from 'svelte-spa-router'

  const menuItems = [
    { href: '/admin/content', label: '콘텐츠 편집', desc: '홈페이지 텍스트 및 섹션 수정' },
    { href: '/admin/inquiries', label: '문의 목록', desc: '고객 문의 내역 확인' },
  ]

  function navigate(e: MouseEvent, href: string) {
    e.preventDefault()
    void push(href)
  }
</script>

<AdminLayout title="대시보드">
  <p class="desc">관리할 항목을 선택하세요.</p>

  <div class="menu-grid">
    {#each menuItems as item}
      <a href={item.href} class="card menu-card" onclick={(e) => navigate(e, item.href)}>
        <h2>{item.label}</h2>
        <p>{item.desc}</p>
      </a>
    {/each}
  </div>
</AdminLayout>

<style>
  .desc {
    color: var(--color-text-muted);
    margin: 0 0 1.5rem;
  }

  .menu-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 1rem;
  }

  .menu-card {
    transition: border-color 0.2s, box-shadow 0.2s;
  }

  .menu-card:hover {
    border-color: var(--color-accent);
    box-shadow: var(--shadow-md);
  }

  .menu-card h2 {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
    color: var(--color-primary);
  }

  .menu-card p {
    margin: 0;
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }
</style>
