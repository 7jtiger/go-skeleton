<script lang="ts">
  import { onMount } from 'svelte'
  import { push } from 'svelte-spa-router'
  import AdminLayout from '../../components/AdminLayout.svelte'
  import { adminLogin } from '../../lib/api'
  import { isTokenValid, setToken } from '../../lib/auth'

  let username = $state('admin')
  let password = $state('')
  let loading = $state(false)
  let error = $state('')

  onMount(() => {
    if (isTokenValid()) {
      push('/admin')
    }
  })

  async function handleSubmit(e: Event) {
    e.preventDefault()
    loading = true
    error = ''
    try {
      const { token } = await adminLogin(username, password)
      setToken(token)
      push('/admin')
    } catch (err) {
      error = err instanceof Error ? err.message : '로그인에 실패했습니다'
    } finally {
      loading = false
    }
  }
</script>

<AdminLayout title="관리자 로그인" requiresAuth={false} showLogout={false}>
  <div class="login-page">
    <form class="login-card card" onsubmit={handleSubmit}>
      <div class="brand">
        <h1>Admin</h1>
      </div>

      <div class="form-group">
        <label for="username">아이디</label>
        <input id="username" type="text" bind:value={username} required />
      </div>

      <div class="form-group">
        <label for="password">비밀번호</label>
        <input id="password" type="password" bind:value={password} required />
      </div>

      {#if error}
        <p class="error">{error}</p>
      {/if}

      <button type="submit" class="btn btn-primary" disabled={loading}>
        {loading ? '로그인 중...' : '로그인'}
      </button>
    </form>
  </div>
</AdminLayout>

<style>
  .login-page {
    min-height: 100vh;
    display: grid;
    place-items: center;
    background: var(--color-surface);
    padding: 1rem;
  }

  .login-card {
    width: min(100%, 400px);
    padding: 2rem;
  }

  .brand {
    text-align: center;
    margin-bottom: 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
  }

  h1 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-text-muted);
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .login-card .btn {
    width: 100%;
    margin-top: 0.5rem;
  }

  .error {
    color: var(--color-danger);
    font-size: 0.9rem;
    margin: 0 0 1rem;
  }
</style>
