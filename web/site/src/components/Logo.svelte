<script lang="ts">
  type Variant = 'light' | 'dark'
  type Size = 'sm' | 'md' | 'lg'

  let {
    variant = 'light',
    size = 'md',
    showText = true,
  }: {
    variant?: Variant
    size?: Size
    showText?: boolean
  } = $props()

  const iconColor = $derived(variant === 'dark' ? '#5EEAD4' : '#14B8A6')
  const textColor = $derived(variant === 'dark' ? '#FFFFFF' : '#0F172A')

  const sizes = {
    sm: { icon: 28, text: '0.95rem', gap: '0.45rem' },
    md: { icon: 36, text: '1.15rem', gap: '0.6rem' },
    lg: { icon: 48, text: '1.5rem', gap: '0.75rem' },
  }

  const s = $derived(sizes[size])
</script>

<span class="logo" style:gap={s.gap} aria-label="LIVEIN">
  <svg
    width={s.icon}
    height={s.icon}
    viewBox="0 0 48 48"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    class="logo-icon"
  >
    <path
      d="M24 4L40.7846 13.5V32.5L24 42L7.21539 32.5V13.5L24 4Z"
      stroke={iconColor}
      stroke-width="2"
      stroke-linejoin="round"
    />
    <path
      d="M15 17H33M15 24H26M15 31H33M26 24V31"
      stroke={iconColor}
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <circle cx="30" cy="17" r="1.5" fill={iconColor} />
    <circle cx="18" cy="31" r="1.5" fill={iconColor} />
  </svg>

  {#if showText}
    <span class="logo-text" style:color={textColor} style:font-size={s.text}>LIVEIN</span>
  {/if}
</span>

<style>
  .logo {
    display: inline-flex;
    align-items: center;
    line-height: 1;
  }

  .logo-text {
    font-weight: 800;
    letter-spacing: 0.14em;
    font-family: 'Pretendard', system-ui, sans-serif;
  }

  .logo-icon {
    flex-shrink: 0;
  }
</style>
