<script lang="ts">
  import { onMount } from 'svelte'
  import AdminLayout from '../../components/AdminLayout.svelte'
  import { adminFetchInquiries } from '../../lib/api'
  import { getToken } from '../../lib/auth'
  import type { Inquiry } from '../../lib/types'

  let inquiries = $state<Inquiry[]>([])
  let loading = $state(true)
  let error = $state('')

  onMount(async () => {
    const token = getToken()
    if (!token) return

    try {
      inquiries = await adminFetchInquiries(token)
      inquiries = [...inquiries].reverse()
    } catch (err) {
      error = err instanceof Error ? err.message : '목록을 불러올 수 없습니다'
    } finally {
      loading = false
    }
  })

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleString('ko-KR')
    } catch {
      return iso
    }
  }
</script>

<AdminLayout title="문의 목록" showBack={true}>
  {#if loading}
    <p>로딩 중...</p>
  {:else if error}
    <p class="error">{error}</p>
  {:else if inquiries.length === 0}
    <p class="empty">접수된 문의가 없습니다.</p>
  {:else}
    <div class="list">
      {#each inquiries as inquiry}
        <article class="card inquiry">
          <div class="top">
            <strong>{inquiry.name}</strong>
            <span class="date">{formatDate(inquiry.createdAt)}</span>
          </div>
          <a href="mailto:{inquiry.email}" class="email">{inquiry.email}</a>
          <p>{inquiry.message}</p>
        </article>
      {/each}
    </div>
  {/if}
</AdminLayout>

<style>
  .list {
    display: grid;
    gap: 1rem;
  }

  .top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.35rem;
  }

  .date {
    font-size: 0.8rem;
    color: var(--color-text-muted);
  }

  .email {
    color: var(--color-accent);
    font-size: 0.9rem;
    display: inline-block;
    margin-bottom: 0.75rem;
  }

  .inquiry p {
    margin: 0;
    color: var(--color-text-muted);
    line-height: 1.7;
    white-space: pre-wrap;
  }

  .empty,
  .error {
    text-align: center;
    padding: 3rem;
    color: var(--color-text-muted);
  }

  .error {
    color: var(--color-danger);
  }
</style>
