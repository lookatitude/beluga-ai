# Content Audit — Website vs Canonical Docs

**Audited:** 2026-04-12
**Canonical total recomputed:** 4+1+6+22+9+13+8+8+5+1+3+1+6+7+3+3+2+1+7 = **110** across **19 categories** — matches `docs/reference/providers.md`.

---

## 1. Summary table

| Severity | File | Line | Current claim | Canonical value | Source ref |
|---|---|---|---|---|---|
| **P0** | `docs/website/src/pages/404.astro` | 41 | "Browse 93 providers" | 110 providers | `docs/reference/providers.md` — total |
| **P0** | `docs/website/src/pages/index.astro` | 72 | STT · TTS · S2S row: "16 providers" | STT(6) + TTS(7) + S2S(3) = **16** — count is correct but list omits "openai tts" and "groq-tts"; groq appears only once in the name string when it is both STT and TTS | `docs/reference/providers.md` § STT, TTS, S2S |
| **P0** | `docs/website/src/pages/index.astro` | 73 | Voice transport · VAD row: "5 providers" (livekit · daily · pipecat · silero · webrtc) | transport(3) + VAD(2) = 5 — count is correct; list is correct | `docs/reference/providers.md` § Voice transport, VAD — **no issue** |
| **P1** | `docs/website/src/content/docs/docs/reference/index.md` | 14 | "All 110 providers across 19 categories" | 110 across 19 — correct; no mismatch | — |
| **P0** | `docs/website/src/components/marketing/LayerStack.astro` | 14 | L05 Orchestration items: "Supervisor · Handoff · Scatter-Gather · Pipeline · Blackboard" | Canonical L5 includes **Router** as a sixth pattern | `docs/architecture/01-overview.md` L5 row; `docs/feature-status.md` `orchestration/` Notes: "Supervisor, Handoff, Scatter-Gather, Pipeline, Blackboard, Router" |
| **P1** | `docs/website/src/pages/index.astro` | 69–70 | Vector stores row lists 12 names (pgvector · pinecone · qdrant · weaviate · milvus · chroma · redis · elasticsearch · mongodb · vespa · turbopuffer · sqlitevec) | Canonical count is 13; **inmemory** is the 13th | `docs/reference/providers.md` § Vector stores |
| **P1** | `docs/website/src/pages/product.astro` | 155 | "Thirteen vector-store backends, nine embedding providers, eight memory stores" | Vector stores: 13 ✓, Embedding: 9 ✓, Memory stores: 8 ✓ — counts are correct; **no issue** | — |
| **P1** | `docs/website/src/pages/product.astro` | 194 | "Six STT providers, seven TTS providers, three speech-to-speech providers" | STT: 6 ✓, TTS: 7 ✓, S2S: 3 ✓ — counts are correct; **no issue** | — |
| **P0** | `docs/website/src/pages/index.astro` | 43–51 | `providerCode` comment "110 providers across 19 categories" | Correct — no issue | — |
| **P0** | `docs/website/src/pages/index.astro` | 234 | Section eyebrow "110 PROVIDERS · 19 CATEGORIES" | Correct — no issue | — |
| **P1** | `docs/website/src/pages/index.astro` | 67 | `providerRows` LLM row: "22 providers" with list of 22 names | Count is 22 ✓; list contains all 22 canonical providers ✓ — no issue | `docs/reference/providers.md` § LLM |
| **P1** | `docs/website/src/pages/index.astro` | 71 | Document loaders row: "8 providers" | Canonical count is 8 ✓; list of 8 names is correct ✓ — no issue | — |
| **P1** | `docs/website/src/pages/index.astro` | 72 | STT · TTS · S2S row: "16 providers" with 14 names listed | Count says 16, but only **14 names** appear: deepgram · whisper · elevenlabs · assemblyai · gladia · groq · cartesia · fish · lmnt · playht · smallest · openai-realtime · gemini-live · amazon-nova. Missing: **smallest** is present; missing provider names are **openai-tts** and **groq-tts** (TTS variants of Groq and OpenAI that are separate from STT Groq/Whisper). The rendered string omits two TTS provider names despite the count being correct. | `docs/reference/providers.md` § TTS: "cartesia · elevenlabs · fish · groq · lmnt · playht · smallest" — 7 providers; STT: "assemblyai · deepgram · elevenlabs · gladia · groq · whisper" — 6 providers |
| **P2** | `docs/website/src/pages/index.astro` | 72 | STT · TTS · S2S row provider name list uses combined 14-name string | For STT+TTS+S2S the combined total is 16, but the name string collapses groq (appears in both STT and TTS) and elevenlabs (appears in both STT and TTS) into single appearances, giving the impression of 14 distinct providers when the count is 16. A reader comparing names to count will find the math wrong. Either the category should be split into rows or the name list should acknowledge that some names appear in multiple sub-categories. | `docs/reference/providers.md` § STT, TTS |
| **P0** | `docs/website/src/pages/404.astro` | 41 | "Browse 93 providers" | 110 | `docs/reference/providers.md` total |

---

## Consolidated finding list (deduped)

### F-01 — P0: `404.astro` stale provider count

**File:** `docs/website/src/pages/404.astro:41`
**Current:** `Browse 93 providers`
**Canonical:** 110 providers

### F-02 — P0: `LayerStack.astro` L5 Orchestration missing "Router"

**File:** `docs/website/src/components/marketing/LayerStack.astro:14`
**Current:** `items: ["Supervisor · Handoff · Scatter-Gather · Pipeline · Blackboard"]`
**Canonical:** `Supervisor · Handoff · Scatter-Gather · Pipeline · Blackboard · Router`
**Why this matters:** `docs/architecture/01-overview.md` lists Router as one of the five orchestration patterns in L5. `docs/feature-status.md` `orchestration/` notes "Supervisor, Handoff, Scatter-Gather, Pipeline, Blackboard, Router". LayerStack is rendered on the homepage, `/product`, and the concepts architecture page — it is the most-viewed architectural diagram on the site.

### F-03 — P1: `index.astro` vector stores name list missing `inmemory`

**File:** `docs/website/src/pages/index.astro:69`
**Current:** "Vector stores — 13 providers — pgvector · pinecone · qdrant · weaviate · milvus · chroma · redis · elasticsearch · mongodb · vespa · turbopuffer · sqlitevec" (12 names)
**Canonical:** 13 providers; the 13th is `inmemory` (from `docs/reference/providers.md` § Vector stores)
**Note:** The count "13" is correct. The name list is one short.

### F-04 — P2: `index.astro` STT·TTS·S2S name list count/name mismatch

**File:** `docs/website/src/pages/index.astro:72`
**Current:** "STT · TTS · S2S — 16 providers — deepgram · whisper · elevenlabs · assemblyai · gladia · groq · cartesia · fish · lmnt · playht · smallest · openai-realtime · gemini-live · amazon-nova" (14 distinct names for 16 providers)
**Canonical:** STT(6): assemblyai, deepgram, elevenlabs, gladia, groq, whisper; TTS(7): cartesia, elevenlabs, fish, groq, lmnt, playht, smallest; S2S(3): gemini, nova, openai. Groq and ElevenLabs appear in both STT and TTS.
**Why:** A reader who counts 14 names and sees "16 providers" is confused. The fix is either (a) annotate shared providers, (b) split the row into three sub-rows, or (c) add a footnote. This is a judgment call — see section 3.

---

## 2. Fixes applicable directly (Ready to apply)

### Fix 1 — `404.astro` line 41

**File:** `docs/website/src/pages/404.astro`

```
old_string: Browse 93 providers
new_string: Browse 110 providers
```

### Fix 2 — `LayerStack.astro` line 14

**File:** `docs/website/src/components/marketing/LayerStack.astro`

```
old_string:   { n: "05", name: "Orchestration", tag: "Multi-agent coordination", items: ["Supervisor · Handoff · Scatter-Gather · Pipeline · Blackboard"] },
new_string:   { n: "05", name: "Orchestration", tag: "Multi-agent coordination", items: ["Supervisor · Handoff · Scatter-Gather · Pipeline · Blackboard · Router"] },
```

### Fix 3 — `index.astro` vector stores name list (line 69)

**File:** `docs/website/src/pages/index.astro`

```
old_string:   ["Vector stores", "13 providers", "pgvector · pinecone · qdrant · weaviate · milvus · chroma · redis · elasticsearch · mongodb · vespa · turbopuffer · sqlitevec"],
new_string:   ["Vector stores", "13 providers", "pgvector · pinecone · qdrant · weaviate · milvus · chroma · redis · elasticsearch · mongodb · vespa · turbopuffer · sqlitevec · inmemory"],
```

---

## 3. Requires judgment

### J-01 — STT·TTS·S2S combined row name/count mismatch (`index.astro:72`)

The row shows "16 providers" but only 14 distinct provider slugs appear in the name string because groq and elevenlabs each appear in two sub-categories (STT and TTS). Options:

**Option A** — Annotate: change the name string to `"deepgram · whisper · elevenlabs (stt+tts) · assemblyai · gladia · groq (stt+tts) · cartesia · fish · lmnt · playht · smallest · openai-realtime · gemini-live · amazon-nova"` — honest but verbose.

**Option B** — Split into three rows: STT (6), TTS (7), S2S (3) — cleaner, but changes the table layout.

**Option C** — Change the count to 14 (unique provider slugs rather than total slots) — technically wrong against the canonical count of 16, but matches what a reader sees. Not recommended.

**Coordinator decision required** before applying.

### J-02 — `index.astro` "openai-tts" not a registered slug

The TTS provider in `docs/reference/providers.md` does not include an "openai" TTS entry — the seven TTS providers are: cartesia, elevenlabs, fish, groq, lmnt, playht, smallest. The `index.astro` STT·TTS·S2S row currently does not list an "openai-tts" entry, so there is no active mismatch here. However, `providers.astro` loads providers dynamically from the content collection tree and would render OpenAI TTS if a file `voice/openai.md` existed. No file does. Flagged in case a future provider addition requires the row to be updated.

---

## 4. Cross-reference index

| Fact verified | Canonical source |
|---|---|
| Total provider count = 110 | `docs/reference/providers.md` — "Total providers: 110 across 19 categories" |
| Total categories = 19 | `docs/reference/providers.md` — 19 section headings |
| LLM providers = 22 | `docs/reference/providers.md` § LLM — **Count: 22** |
| Embedding providers = 9 | `docs/reference/providers.md` § Embedding — **Count: 9** |
| Vector store providers = 13 (including inmemory) | `docs/reference/providers.md` § Vector stores — **Count: 13** |
| Document loader providers = 8 | `docs/reference/providers.md` § Document loaders — **Count: 8** |
| Memory store providers = 8 | `docs/reference/providers.md` § Memory stores — **Count: 8** |
| Guard providers = 5 | `docs/reference/providers.md` § Guard — **Count: 5** |
| STT providers = 6 | `docs/reference/providers.md` § STT — **Count: 6** |
| TTS providers = 7 | `docs/reference/providers.md` § TTS — **Count: 7** |
| S2S providers = 3 | `docs/reference/providers.md` § S2S — **Count: 3** |
| Voice transport providers = 3 | `docs/reference/providers.md` § Voice transport — **Count: 3** |
| VAD providers = 2 | `docs/reference/providers.md` § VAD — **Count: 2** |
| Workflow engine providers = 6 | `docs/reference/providers.md` § Workflow engines — **Count: 6** |
| Observability exporters = 4 | `docs/reference/providers.md` § Observability — **Count: 4** |
| Reasoning strategies = 8 | `docs/architecture/06-reasoning-strategies.md` lines 14–33 |
| Strategy names | ReAct, Reflexion, Self-Discover, MindMap, Tree-of-Thought, Graph-of-Thought, LATS, Mixture-of-Agents |
| L5 Orchestration patterns include Router | `docs/architecture/01-overview.md` L5 description; `docs/feature-status.md` `orchestration/` Notes |
| L5 item list in LayerStack | `docs/website/src/components/marketing/LayerStack.astro:14` |
| 7-layer architecture | `docs/architecture/01-overview.md` — The 7-layer model |
| L1 Foundation packages | core · schema · config · o11y — `docs/architecture/01-overview.md` |
| L2 Cross-cutting includes Sandbox | `docs/architecture/01-overview.md` "resilience · auth · audit · cost · state · sandbox · workflow" |
| LayerStack L2 includes Sandbox | `docs/website/src/components/marketing/LayerStack.astro:17` items: "Resilience · Auth · Audit · Cost · State · Sandbox · Workflow" — correct |
| 4 extensibility mechanisms | interface / registry / hooks / middleware — `docs/architecture/03-extensibility-patterns.md` |
| Website claims 4 mechanisms | `docs/website/src/content/docs/docs/concepts/extensibility.md:6` "four extension mechanisms" — correct |
| `workflow/` status = Stable | `docs/feature-status.md` § Stable — "Durable workflow — workflow/" |
| `voice/` status = Stable | `docs/feature-status.md` § Stable — all voice sub-packages |
| Guard provider count = 5 | `docs/feature-status.md` § Stable "5 guard providers: Lakera, NeMo, LLM Guard, Guardrails AI, Azure AI Content Safety" |
| Memory store count = 8 (not 9) | `docs/feature-status.md` § Stable: "9 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly)" — **feature-status says 9 but `docs/reference/providers.md` lists 8 named providers** — see F-05 below |

---

### F-05 — P0: `feature-status.md` memory store count disagrees with `providers.md`

**This is a canonical source conflict requiring coordinator resolution.**

- `docs/feature-status.md:47` says: `9 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly)`
- Counting the parenthetical list: inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly = **8 providers**
- `docs/reference/providers.md` § Memory stores lists 8 entries with **Count: 8**

The number "9" in `feature-status.md` appears to be a typo (8 names are listed after the claim of 9). The website (`product.astro:155`) says "eight memory stores" — which matches `providers.md` and the actual list in `feature-status.md`.

**Canonical value:** 8 memory store providers.
**Mismatch:** `feature-status.md:47` says "9 store providers" but lists 8.
**Fix required in:** `docs/feature-status.md` line 47, not the website.

| Severity | File | Line | Current claim | Canonical value | Source ref |
|---|---|---|---|---|---|
| **P0** | `docs/feature-status.md` | 47 | "9 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly)" | 8 providers (the parenthetical list has 8 names) | `docs/reference/providers.md` § Memory stores — Count: 8 |

**Ready to apply:**

**File:** `docs/feature-status.md`

```
old_string: | Memory | `memory/` | 3-tier MemGPT model; 9 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly) |
new_string: | Memory | `memory/` | 3-tier MemGPT model; 8 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly) |
```

---

## Audit scope confirmed clean

The following claims were verified and found **correct** — no action needed:

| Claim | Location | Verdict |
|---|---|---|
| "110 providers" total | `index.astro:104`, `index.astro:119`, `index.astro:234`, `product.astro:230`, `compare.astro:69` | Correct |
| "19 categories" | `index.astro:234`, `compare.astro:69` | Correct |
| "8 built-in" reasoning strategies with full list | `index.astro:181–184`, `compare.astro:37–38` | Correct; list matches `docs/architecture/06-reasoning-strategies.md` |
| "Four rings" extensibility | `concepts/extensibility.md:11`, `docs/architecture/03-extensibility-patterns.md` | Correct |
| `workflow/` marked Stable | `enterprise.astro:44`, `workflow-durability.md` entire file | Correct per `feature-status.md` § Stable |
| 6 workflow backends | `compare.astro:53`, `enterprise.astro:44` | Correct per `providers.md` Count: 6 |
| 17 packages ship `WithTracing()` | `index.astro:195`, `compare.astro:45`, `concepts/extensibility.md:184` | Correct per `docs/architecture/03-extensibility-patterns.md` |
| 7-layer architecture claim | All pages referencing LayerStack | Correct (7 layers rendered) |
| L1 Foundation: core · schema · config · o11y | `LayerStack.astro:18` | Correct |
| L2 Cross-cutting includes Sandbox | `LayerStack.astro:17` | Correct |
| L3 Capability includes HITL | `LayerStack.astro:16` | Correct |
| L6 Agent runtime: Runner · Agent · Executor · Planner · Team | `LayerStack.astro:13` | Correct |
| L7 Application items | `LayerStack.astro:12` | Correct |
| LLM: 22 providers | `index.astro:67`, `product.astro:134`, `feature-status.md:45` | Correct |
| Embedding: 9 providers | `index.astro:69`, `product.astro:155` | Correct |
| Vector store: 13 providers (count) | `index.astro:69`, `product.astro:155` | Correct |
| Memory stores: 8 | `product.astro:155` (website) | Correct (website is right; `feature-status.md` has the typo) |
| Guard: 5 providers with correct names | `enterprise.astro:34`, `feature-status.md:57` | Correct |
| STT: 6, TTS: 7, S2S: 3 (individual counts) | `product.astro:194` | Correct |
| Voice transport: LiveKit, Daily, Pipecat | `product.astro:197` | Correct |
| VAD: Silero and WebRTC | `product.astro` (implicit) | Correct |
| No `.md` path references without real links in `concepts/*.md` | All concepts pages reviewed | No bare `.md` paths found; all cross-references use anchor hrefs |
| `resilience.md` — no Planned features claimed as shipped | `guides/production/resilience.md` | Clean |
| `workflow-durability.md` — `workflow/` described as Stable | `guides/production/workflow-durability.md` | Correct per `feature-status.md` |
