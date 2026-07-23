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
  commandPath="rportfwd add"
  title="Add Reverse Port Forward"
  description="Open a listener <strong>on the implant</strong> that forwards connections back <strong>to the operator/teamserver</strong>. Use this to expose an operator-side service (Responder, Impacket <code>ntlmrelayx</code>, a fake update server) to the target's network."
  example="Example: implant binds <code>0.0.0.0:4444</code>, forwards to <code>127.0.0.1:8080</code> on the operator. Anyone on the target LAN that hits the implant on port 4444 lands on your local <code>:8080</code>."
  bindLabel="Implant listen (on target machine)"
  bindPlaceholder="0.0.0.0"
  bindPortPlaceholder="4444"
  bindDescription="Interface + port the implant listens on. 0.0.0.0 = all interfaces."
  bindRequired={true}
  bindDefault="0.0.0.0:4444"
  remoteLabel="Forward to (on operator/server)"
  remotePlaceholder="127.0.0.1"
  remotePortPlaceholder="4444"
  remoteDescription="Where connections are relayed to, from the server's perspective"
  remoteRequired={true}
  remoteDefault="127.0.0.1:4444"
/>
