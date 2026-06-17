<svelte:options runes={true} />

<script lang="ts">
  import { onMount } from 'svelte'
  import AdminLayout from '../../components/AdminLayout.svelte'
  import { adminUpdateContent, fetchContent } from '../../lib/api'
  import { getToken } from '../../lib/auth'
  import type { Content } from '../../lib/types'

  let content = $state<Content | null>(null)
  let activeTab = $state('hero')
  let saving = $state(false)
  let toast = $state<{ type: 'success' | 'error'; msg: string } | null>(null)

  const tabs = [
    { id: 'hero', label: 'Hero' },
    { id: 'services', label: '서비스' },
    { id: 'solutions', label: '솔루션' },
    { id: 'chat', label: '채팅' },
    { id: 'trust', label: '신뢰' },
    { id: 'process', label: '프로세스' },
    { id: 'about', label: '소개' },
    { id: 'contact', label: '연락처' },
  ]

  onMount(async () => {
    content = await fetchContent()
  })

  function showToast(type: 'success' | 'error', msg: string) {
    toast = { type, msg }
    setTimeout(() => (toast = null), 3000)
  }

  async function save() {
    if (!content) return
    const token = getToken()
    if (!token) return

    saving = true
    try {
      content = await adminUpdateContent(content, token)
      showToast('success', '저장되었습니다')
    } catch (err) {
      showToast('error', err instanceof Error ? err.message : '저장 실패')
    } finally {
      saving = false
    }
  }

  function addService() {
    if (!content) return
    content.services = [
      ...content.services,
      { id: `svc-${Date.now()}`, title: '', description: '', icon: 'code' },
    ]
  }

  function removeService(i: number) {
    if (!content) return
    content.services = content.services.filter((_, idx) => idx !== i)
  }

  function addSolution() {
    if (!content) return
    content.solutions = [
      ...content.solutions,
      { id: `sol-${Date.now()}`, name: '', status: 'development', summary: '' },
    ]
  }

  function removeSolution(i: number) {
    if (!content) return
    content.solutions = content.solutions.filter((_, idx) => idx !== i)
  }
</script>

<AdminLayout title="콘텐츠 편집" showBack={true}>
  {#snippet actions()}
    <button type="button" class="btn btn-primary" onclick={save} disabled={saving || !content}>
      {saving ? '저장 중...' : '저장'}
    </button>
  {/snippet}

  {#if content}
      <div class="tabs">
        {#each tabs as tab}
          <button
            class="tab"
            class:active={activeTab === tab.id}
            onclick={() => (activeTab = tab.id)}
          >
            {tab.label}
          </button>
        {/each}
      </div>

      {#if activeTab === 'hero'}
        <div class="card">
          <div class="form-group">
            <label>제목</label>
            <input bind:value={content.hero.title} />
          </div>
          <div class="form-group">
            <label>부제목</label>
            <textarea bind:value={content.hero.subtitle}></textarea>
          </div>
          <div class="form-group">
            <label>CTA Primary</label>
            <input bind:value={content.hero.ctaPrimary} />
          </div>
          <div class="form-group">
            <label>CTA Secondary</label>
            <input bind:value={content.hero.ctaSecondary} />
          </div>
        </div>
      {:else if activeTab === 'services'}
        <div class="card">
          {#each content.services as service, i}
            <div class="item-block">
              <div class="form-group">
                <label>제목</label>
                <input bind:value={service.title} />
              </div>
              <div class="form-group">
                <label>설명</label>
                <textarea bind:value={service.description}></textarea>
              </div>
              <div class="form-group">
                <label>아이콘 (code/message/puzzle/shield)</label>
                <input bind:value={service.icon} />
              </div>
              <button class="btn btn-outline danger" onclick={() => removeService(i)}>삭제</button>
            </div>
          {/each}
          <button class="btn btn-outline" onclick={addService}>서비스 추가</button>
        </div>
      {:else if activeTab === 'solutions'}
        <div class="card">
          {#each content.solutions as solution, i}
            <div class="item-block">
              <div class="form-group">
                <label>이름</label>
                <input bind:value={solution.name} />
              </div>
              <div class="form-group">
                <label>상태 (active/development)</label>
                <input bind:value={solution.status} />
              </div>
              <div class="form-group">
                <label>요약</label>
                <textarea bind:value={solution.summary}></textarea>
              </div>
              <button class="btn btn-outline danger" onclick={() => removeSolution(i)}>삭제</button>
            </div>
          {/each}
          <button class="btn btn-outline" onclick={addSolution}>솔루션 추가</button>
        </div>
      {:else if activeTab === 'chat'}
        <div class="card">
          <div class="form-group">
            <label>제목</label>
            <input bind:value={content.chatService.title} />
          </div>
          <div class="form-group">
            <label>설명</label>
            <textarea bind:value={content.chatService.description}></textarea>
          </div>
          <div class="form-group">
            <label>기능 (줄바꿈으로 구분)</label>
            <textarea
              value={content.chatService.features.join('\n')}
              oninput={(e) => {
                content!.chatService.features = e.currentTarget.value.split('\n').filter(Boolean)
              }}
            ></textarea>
          </div>
          <div class="form-group">
            <label>기술 스택 (줄바꿈으로 구분)</label>
            <textarea
              value={content.chatService.techStack.join('\n')}
              oninput={(e) => {
                content!.chatService.techStack = e.currentTarget.value.split('\n').filter(Boolean)
              }}
            ></textarea>
          </div>
        </div>
      {:else if activeTab === 'trust'}
        <div class="card">
          {#each content.trust.metrics as metric}
            <div class="item-block inline">
              <div class="form-group">
                <label>라벨</label>
                <input bind:value={metric.label} />
              </div>
              <div class="form-group">
                <label>값</label>
                <input bind:value={metric.value} />
              </div>
            </div>
          {/each}
          <div class="form-group">
            <label>배지 (줄바꿈으로 구분)</label>
            <textarea
              value={content.trust.badges.join('\n')}
              oninput={(e) => {
                content!.trust.badges = e.currentTarget.value.split('\n').filter(Boolean)
              }}
            ></textarea>
          </div>
        </div>
      {:else if activeTab === 'process'}
        <div class="card">
          {#each content.process as step}
            <div class="item-block inline">
              <div class="form-group">
                <label>단계</label>
                <input type="number" bind:value={step.step} />
              </div>
              <div class="form-group">
                <label>제목</label>
                <input bind:value={step.title} />
              </div>
              <div class="form-group">
                <label>설명</label>
                <input bind:value={step.description} />
              </div>
            </div>
          {/each}
        </div>
      {:else if activeTab === 'about'}
        <div class="card">
          <div class="form-group">
            <label>소개</label>
            <textarea bind:value={content.about.description}></textarea>
          </div>
          <div class="form-group">
            <label>미션</label>
            <textarea bind:value={content.about.mission}></textarea>
          </div>
          <div class="form-group">
            <label>비전</label>
            <textarea bind:value={content.about.vision}></textarea>
          </div>
        </div>
      {:else if activeTab === 'contact'}
        <div class="card">
          <div class="form-group">
            <label>이메일</label>
            <input bind:value={content.contact.email} />
          </div>
          <div class="form-group">
            <label>주소</label>
            <input bind:value={content.contact.address} />
          </div>
          <div class="form-group">
            <label>전화</label>
            <input bind:value={content.contact.phone} />
          </div>
        </div>
      {/if}
  {:else}
    <p>로딩 중...</p>
  {/if}
</AdminLayout>

{#if toast}
  <div class="toast" class:success={toast.type === 'success'} class:error={toast.type === 'error'}>
    {toast.msg}
  </div>
{/if}

<style>
  .item-block {
    padding-bottom: 1.5rem;
    margin-bottom: 1.5rem;
    border-bottom: 1px solid var(--color-border);
  }

  .item-block.inline {
    display: grid;
    grid-template-columns: 80px 1fr 1fr;
    gap: 1rem;
    align-items: end;
  }

  .danger {
    color: var(--color-danger);
    border-color: var(--color-danger);
  }

  @media (max-width: 768px) {
    .item-block.inline {
      grid-template-columns: 1fr;
    }
  }
</style>
