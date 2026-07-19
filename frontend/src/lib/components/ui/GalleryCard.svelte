<script>
  let { src = '', alt = '', title = '', onclick } = $props()
</script>

<button
  type="button"
  class="flex flex-col overflow-hidden rounded-lg border border-line bg-chrome cursor-pointer transition-transform duration-200 hover:-translate-y-0.5 hover:shadow-lg hover:border-brand p-0 text-inherit font-inherit text-left"
  {onclick}
>
  <img {src} {alt} class="w-full h-38 object-cover bg-black" />
  <span class="px-2 py-2 text-sm text-fg text-center truncate">{title}</span>
</button>
