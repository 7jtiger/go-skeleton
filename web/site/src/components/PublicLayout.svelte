<script lang="ts">
  import Header from '../components/Header.svelte'
  import Footer from '../components/Footer.svelte'
  import type { Content } from '../lib/types'

  let { children, content }: { children?: () => unknown; content?: Content | null } = $props()
</script>

<div class="layout">
  <Header />
  <main>
    {@render children?.()}
  </main>
  <Footer contact={content?.contact} />
</div>

<style>
  .layout {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  main {
    flex: 1;
  }
</style>
