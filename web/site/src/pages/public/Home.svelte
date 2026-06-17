<script lang="ts">
  import { onMount } from 'svelte'
  import PublicLayout from '../../components/PublicLayout.svelte'
  import HeroSection from '../../components/sections/HeroSection.svelte'
  import ServicesSection from '../../components/sections/ServicesSection.svelte'
  import SolutionsSection from '../../components/sections/SolutionsSection.svelte'
  import ChatSection from '../../components/sections/ChatSection.svelte'
  import TrustSection from '../../components/sections/TrustSection.svelte'
  import ProcessSection from '../../components/sections/ProcessSection.svelte'
  import ContactCta from '../../components/sections/ContactCta.svelte'
  import { fetchContent } from '../../lib/api'
  import type { Content } from '../../lib/types'

  let content = $state<Content | null>(null)
  let loading = $state(true)
  let error = $state('')

  onMount(async () => {
    try {
      content = await fetchContent()
    } catch (e) {
      error = e instanceof Error ? e.message : '콘텐츠를 불러올 수 없습니다'
    } finally {
      loading = false
    }
  })
</script>

<PublicLayout {content}>
  {#if loading}
    <div class="loading container">로딩 중...</div>
  {:else if error}
    <div class="error container">{error}</div>
  {:else if content}
    <HeroSection hero={content.hero} />
    <ServicesSection services={content.services} />
    <SolutionsSection solutions={content.solutions} />
    <ChatSection chatService={content.chatService} />
    <TrustSection trust={content.trust} />
    <ProcessSection process={content.process} />
    <ContactCta />
  {/if}
</PublicLayout>

<style>
  .loading,
  .error {
    padding: 6rem 0;
    text-align: center;
    color: var(--color-text-muted);
  }

  .error {
    color: var(--color-danger);
  }
</style>
