<script>
  let {
    gap = 'sm',
    align = 'center',
    justify = 'start',
    wrap = true,
    class: className = '',
    children,
    ...restProps
  } = $props()

  const gapClasses = {
    none: 'gap-0',
    xs: 'gap-1',     // 4px
    sm: 'gap-2',     // 8px
    md: 'gap-3',     // 12px
    lg: 'gap-4',     // 16px
    xl: 'gap-6',     // 24px
    '2xl': 'gap-8',   // 32px
  }

  const alignClasses = {
    start: 'items-start',
    center: 'items-center',
    end: 'items-end',
    baseline: 'items-baseline',
    stretch: 'items-stretch',
  }

  const justifyClasses = {
    start: 'justify-start',
    center: 'justify-center',
    end: 'justify-end',
    between: 'justify-between',
    around: 'justify-around',
    evenly: 'justify-evenly',
  }
</script>

<div
  class="flex {wrap ? 'flex-wrap' : 'flex-nowrap'} {gapClasses[gap] || ''} {alignClasses[align] || ''} {justifyClasses[justify] || ''} {className}"
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</div>
