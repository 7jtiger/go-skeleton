<script lang="ts">
  import { onMount } from 'svelte'

  let { children }: { children?: () => unknown } = $props()
  let el: HTMLElement
  let visible = $state(false)

  onMount(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          visible = true
          observer.disconnect()
        }
      },
      { threshold: 0.15 }
    )
    observer.observe(el)
    return () => observer.disconnect()
  })
</script>

<div bind:this={el} class="fade-in" class:visible>
  {@render children?.()}
</div>
