# Gateway library composition

This binding-owned companion for issue #64 advances direction 4 of #59. It
defines the composition, factory and lifecycle requirements for an in-process
Matter library host on the standalone bridge. It documents the AA17 gateway
specification; it does not assert that the host, its launcher, or a production
SDK backend is running. The in-process-versus-separate-process strategy decision
remains owner work outside this repository.

## Adoption and the current boundary

The owner has adopted go-fabric as the SDK, using its Matter 1.6.1 data model,
and the 0.7 Matter bridge ships with development attestation, uncertified (see
[attestation](attestation.md)). Matter 1.7-only elements are deferred.
Matter northbound is `matterreg`; the Home Assistant add-on build carries no
Matter code, while a separately selected development build may include it.

Support is declared from the specification mapping. The committed lock
[`sdk/matter-sdk.lock.json`](../../sdk/matter-sdk.lock.json) pins the go-fabric
`module`, `release_tag` and `checksum`, and the
[readiness evaluator](../../runtime/production_readiness.py) checks that pin
against the lock, the go-fabric build record, the accepted composition profile
and the mapping the lock pins. Admission is from that specification mapping and
the pinned accepted composition profile. A bench, hosted, fixture or
runtime-qualification run is a recorded level and does not hold a mapped
endpoint or a launcher. Nothing here is stable before Helianthus 1.0.

The committed lock keeps `mapping` and `accepted_runtime_composition_profile_sha256`
`null`; the first composition and its mapping pin are written by the first
complete endpoint. Until a reviewed profile and mapping pin exist, the evaluator
gives a reason and permits nothing. A lock whose module, release tag or checksum
differs from the committed pin is refused.

Source provisioning and consumer pinning are separate boundaries. The real
factory link and consumer pin are owned by the AA12/AA30 work; this companion
establishes neither.

## Enablement inputs

Enablement has two independent inputs:

- standalone configured enablement, `OutputBindings.Matter`, which is effective
  without any flag; and
- the explicit `-matter-library-enabled` switch, whose default is false.

The false flag default does not win over a configured standalone enablement.
The flag is an independent enablement input that supports the selected
development launch; it is not a suppression of standalone configuration. The
Disabled state is neither input enabled.

## Managed-driver factory contract

The public managed-driver constructor is
`github.com/Project-Helianthus/helianthus-matterreg/driver.NewFactory`. It is
implemented by the bridge node work that follows this page. The gateway
composition imports only this constructor, in the exact operator-approved
profile file.

The contract is the zero-argument `func NewFactory() *factory`. The returned
concrete factory implements `semreg/v1/driver.Factory`. It also owns
`BindFlags(*flag.FlagSet) func() (enabled bool, options semdriver.Config)` for
the enabled profile. `BindFlags` binds on the existing pre-parse flag set and
reads the returned ordinary enablement and opaque options after parse. It
installs exactly two options:

- `-matter-library-enabled`, the explicit enablement switch (existing default false); and
- `-matter-endpoint-store`, the path of the persistent endpoint and fabric state store.

Standalone `OutputBindings.Matter` stays effective and enables the bridge when no
flag is passed. Official nomatter registers neither option. Direct construction
and pure `Plan(Config)` remain usable in a second host without any flag setup.
The managed lifecycle stays in process.

## Composition states and factory boundary

The following are contract states, not newly defined MCP or GraphQL fields.

| State | Required behavior | Allocation and evidence limits |
| --- | --- | --- |
| Disabled | Neither enablement input is enabled: standalone configured enablement is off and the explicit switch is false. | Zero binding factory calls, SDK allocations and fabric-store allocations. No launcher, listener, commissioning or report dispatch. |
| Unavailable | An enablement input is on, but the evaluator gives a reason: the lock, mapping pin, profile pin, build record or composition is absent or mismatched. Expose the reason. | Zero factory calls and zero SDK or store allocation; no fallback sample, permissive adapter or automatic retry into production. |
| Admitted | The evaluator permits: the pinned mapping, the accepted composition profile and the matching go-fabric module, release tag and checksum agree. | Allocation is the caller's bounded step. A fixture, bench, hosted or runtime-qualification run is a recorded level and blocks nothing. |

AA17 specifies a narrow injected binding factory and a shared snapshot and
evaluation port. These are requirements, not invented go-fabric API names. AA27
owns the separate gateway entry-point integration. No default feed change,
native write route or Portal publication follows from composing the host.

The required ordering is:

1. Resolve both enablement inputs before constructing the binding or its store.
   Standalone configured enablement is effective without a flag; the false flag
   default does not suppress it.
2. When enabled, validate the lock's go-fabric module, release tag and checksum
   against the build record and the composition, with the accepted profile pin
   and the mapping pin set. Check source, binding and native-driver generations
   and the lifecycle token; stale or mismatched inputs cannot pass.
3. Apply the binding's reviewed admission decision before any production
   allocation. Admission is from the specification mapping and the pinned
   accepted composition profile. Caller-supplied metadata, a successful fixture
   or a successful build cannot replace the pins or the mapped target facts.
4. Only an admitted integration may invoke the real factory, allocate SDK
   resources and start reporting within its declared limits. The launcher is
   the bridge's managed lifecycle, which stays in process.

The required Descriptor, Power Topology and Electrical Power Measurement facts
cannot be inferred from optional ActiveCurrent, a sample endpoint or library
defaults. Any later change of the pinned go-fabric release or mapping needs its
own reviewed pins and build record.
Qualification and admission are evidence boundaries. This adds no principal,
credential or authorization gate before 0.9; authentication and authorization
remain 0.11 roadmap work.

## Shared semantic snapshots

The host consumes the shared semantic snapshot and evaluation port through the
AA03 projector. It must not round-trip through HTTP, create a second semantic
truth store, import vendor decoding or reinterpret raw protocol values. The
same detached snapshot and evidence decision must underpin equivalent MCP,
GraphQL and Matter projections; preserve eBUS MCP compatibility and GraphQL
parity. This companion changes no public envelope or namespace.

Preserve value, availability, freshness, confidence, qualification and native
provenance as independent facts. Proven, Hypothesis and Unknown describe the
grade of a claim; the grade is recorded and never denies an otherwise valid row.
Protocol facts require the public native documentation contract first, then
accepted semantic evidence and the public Matter target mapping. This page
introduces no protocol facts or mappings. See the
[inventory completeness contract](../target-profile/inventory-completeness.md).

Disabled, unavailable, evidence loss and teardown must preserve native evidence
and partial last-known-good semantic state. Retained values must not become
fresh through restart, retry or projection. Withholding one affected capability
must preserve independently valid records and must not select a substitute
target. Reporting cannot upgrade an unknown fact.

## Generation-fenced lifecycle

The [runtime qualification scenarios](../go-fabric-runtime-qualification.md)
define the required asynchronous and resource boundaries. Each successful start
has a fresh generation; callbacks must also match their capability/composition
key and operation token. Source, binding and native-driver generations remain
independent checks. An old success or failure cannot publish into a successor
generation, restore a retired registration or overwrite newer evidence.

Stop first fences new work and callbacks, cancels queued work and drains owned
in-flight operations within a declared finite budget. Join tasks and release
registrations and resources before declaring stop complete or allowing restart.
A stale callback may release only its own bookkeeping exactly once. Failure to
quiesce keeps the old generation fenced and prohibits successful restart while
its resources remain unsafe. Successful stop leaves zero owned tasks,
registrations, queued items and in-flight work. Repeated stop is idempotent;
duplicate start cannot allocate another running host.

Report dispatch and receipt are distinct. Cancellation after dispatch may leave
delivery unknown; do not claim retraction, exactly-once delivery or automatically
replay an ambiguous report after restart. Preserve evidence and loss disposition.
Fixture budgets and fake-sink acknowledgements are not library resource limits
or proof of controller receipt.

Endpoint identity and selection persistence follow the Portal lifecycle contract
and the endpoint store. Endpoint numbers are stable and are never reused; the
store has no resource-count cap, and the only maximum is the 16-bit endpoint
id. Do not fabricate identity or fabric material from selection, cached values or
fixture tokens. Corrupt or ambiguous restoration withholds affected publication
while retaining valid unaffected records. No automatic commissioning or rejoin
occurs on start, restart, enable, evidence recovery or fabric removal. Any future
real commissioning still requires the existing action-time operator confirmation
in [the safety rules](../../AGENTS.md). No live action is authorized here.

## Evidence and remaining work

Offline acceptance of this document is manual comparison against the lifecycle
rules, runtime qualification scenarios and AA17 specification, plus a
standard-library local-link check. It is documentation evidence only. AA17
calls for isolated controls proving disabled and incomplete-evidence zero
factory and allocation behavior, rejection of stale callbacks, joined teardown
and three start and stop cycles with zero owned resources. Those are required
code checks, not results claimed by this page.

Focused synthetic fixtures, in-memory loopback and injected logical time are
recorded levels. They do not gate the mapped endpoint. Linux arm64 support, Pi
performance, latency and physical behavior are recorded as Unknown until a
matching run exists. Transport conformance, live-fabric smoke, interoperability
and Matter certification are not established. There is no real-bus transmit path
or production eBUS arming switch introduced by this documentation; answering as a
slave on a real bus would be a live write and is outside this work.

Real factory linking, consumer pin review, the first complete endpoint and its
mapping pin, and independent integration review remain required. Selective Portal
publication, persistence and UI implementation and closure of #4 remain outside
this slice. This does not close the #59 umbrella or establish release or
physical acceptance.
