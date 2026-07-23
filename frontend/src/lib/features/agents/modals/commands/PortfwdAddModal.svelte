<script>
  import PortForwardModal from '../PortForwardModal.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()
</script>

<PortForwardModal
  bind:open
  {onexecute}
  {onclose}
  {initialValues}
  commandPath="portfwd add"
  title="Add Port Forward"
  description="Forward traffic <strong>from your operator machine &rarr; through the implant &rarr; to a target service</strong>. Reachable services on the implant's network segment become reachable from your operator machine's <code>localhost</code>."
  example="Example: <code>127.0.0.1:8080</code> &rArr; <code>10.0.0.5:80</code> &mdash; pointing your browser at <code>http://127.0.0.1:8080</code> hits the internal web server at <code>10.0.0.5:80</code> from the implant's perspective."
  bindLabel="Local bind (on operator)"
  bindPlaceholder="127.0.0.1"
  bindPortPlaceholder="8080"
  bindDescription="Where you connect from"
  bindDefault="127.0.0.1:8080"
  remoteLabel="Remote target (through implant)"
  remotePlaceholder="10.0.0.5"
  remotePortPlaceholder="80"
  remoteDescription="Where the traffic actually goes, as seen from the implant"
  remoteRequired={true}
  remoteDefault=""
/>
