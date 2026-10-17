# Drift: Warmth, Yearning and Safety

Oct 16, 2026 · @Vedant

## Summary

Build the warm, red-coral world of *Her* for everyone, and let each person choose their own glow. Do not give women a red default. A red marker that is automatic for women shows strangers who to swarm, which is the main reason women leave stranger-chat apps.

What we can implement, in order:

1. **Safety first (week 1):** gender colour private to strangers, knock-only mode, per-sender limit on new chats, blurred images until accepted.
2. **The aura (week 2):** an opt-in glow ring around your face, picked from a *Her*-inspired set. Anyone can choose red, so it means "I chose this", not "I am a woman".
3. **The reveal ladder (weeks 3-4):** face, then conversation, then voice, then photo. Each step is mutual. This is the slow burn.
4. **Faces v3 (weeks 4-5):** softer, warmer face style as a new version beside v1 and v2, so no existing face changes.
5. **Yearning features (weeks 5-8):** remembered conversations, "pick this up tomorrow", letters, conversation prompts.

## Research basis

Colour sets mood reliably; colour as a sexual signal does not replicate. Figures below are from memory and approximate; verify before citing externally.

| Finding | Strength | What we take from it |
| --- | --- | --- |
| Red makes women look more attractive to men (Elliot and Niesta, 2008) | Weak: a large replication (Lehmann et al., 2018) found little or no effect | Do not build on it |
| Warm, dim, low-blue light reads as intimate and suits night use | Strong: blue light suppresses melatonin; warm light lowers arousal | Keep the dark, amber, cream base |
| A brand colour signals who a space is for | Industry case: Bumble chose yellow to read warm and safe to women | Warmth over heat |
| Anticipation drives reward more than the reward itself (Schultz, dopamine studies) | Strong | Reveal in mutual steps |
| Unfinished things stay on the mind (Zeigarnik effect) | Moderate | A chat can end on "tomorrow?" |
| Loneliness is widespread among young adults (WHO 2023 called it a global health concern) | Strong | Being remembered matters more than being matched |

**What *Her* teaches.** Production designer K.K. Barrett gave the film a warm red-coral and pastel palette and kept out cold blue. The red is on Theodore, a man, and on the world around him. It signals warmth and closeness, not sex. That is the palette to borrow.

**What Ana Huang teaches.** Her mostly female readers stay for yearning inside safety. Tension is high, but the heroine is never unsafe with the hero. The key beats are a slow reveal, being noticed in detail, banter, and a pause before the next chapter.

## The red-for-women question

Decision: no red by default for women; red is one aura anyone can pick. Three reasons:

- **It makes women targets.** Today `--g-female #FF8FC7` already colours women's names pink in every list (`web/style.css` lines 19-24, `web/app.js` sets `data-gender` in 9 places). A red ring would be louder still. On stranger-chat apps, women who are easy to spot get flooded with openers in their first minute and leave.
- **It rests on a finding that does not replicate.** The "red means available" effect shrank to near zero in large studies.
- **It reads as a label, not a choice.** A woman who picks red is expressing herself. A woman who is given red is marked.

What to do instead:

- Stop colouring names by gender for strangers. Show gender as text on the profile only, with a setting to hide it from people you have not talked to.
- Keep gender colours inside a chat you have both opened, if users like them.
- Offer red ("Ember") as one aura among several, for any gender.

## The aura

The aura is a soft glow ring around a person's face that they choose. It is off by default and the same set is offered to everyone.

| Aura | Hex | Mood |
| --- | --- | --- |
| Ember | #E8604C | *Her* red-coral; warm, open |
| Rose | #C76B7E | Tender, close |
| Lantern | #FFB547 | The app's amber; friendly |
| Peach | #F4A988 | Soft, playful |
| Dusk | #9C7BD8 | Dreamy, night owl |
| Sea | #6FBFAE | Calm, listener |

Implementation:

- **Data:** migration `004_aura.sql` adds `aura VARCHAR(16) NULL` to users. An allowlist of the six keys lives in `internal/user`; anything else is rejected with `400`.
- **API:** `PATCH /me` accepts `aura` (key or `null`). `Card` gains `aura`. The existing `presence` update carries it, so no new event.
- **Render:** CSS only, so the face images stay cached for a year. `.face[data-aura="ember"] { box-shadow: 0 0 0 2px var(--aura-ember), 0 0 12px var(--aura-ember-glow); }`. A slow 4 s breathing pulse, disabled under `prefers-reduced-motion`.
- **Picker:** six swatches in the profile sheet, each with its mood word.
- **Tests:** allowlist validation in `internal/user`, card serialisation, and a hub test that presence carries the aura.

Optional later: unlock extra auras through good conduct (chats that last, replies received), never through messages sent.

## Faces v3

Add a warmer face style as `/avatar/v3`, beside v1 (pixel animals) and v2 (society marks). No existing face changes. The seed format is append-only, and v1 must match `internal/avatar/testdata/golden.json` pixel for pixel.

What changes in v3, and why:

- **Warmer backgrounds.** Today's v2 backgrounds are near-black and the glows include Ghost and Venom. v3 uses the *Her* range: coral, peach, rose, dusk, warm cream.
- **Softer, more open eyes.** Swap v2's hollow, slit and void eyes for open, sleepy and smiling ones. A face that looks approachable gets more replies than one that looks guarded, for every gender.
- **Blush and small expressions.** v1 already has `blush`; make it common in v3, plus a rare shy-smile or wink frame for the idle animation.
- **Faces are never assigned by gender.** The seed is the normalised name plus the style the person picked (next section), never their gender field. This keeps the protection from the previous section.

How to build it:

1. Prototype in the browser first, as with v1, and save a new `testdata/golden_v3.json`.
2. Reuse the v2 mesh (`maskAnimals`, `maskFacets`) and add a new palette and eye set. Append new seed slots; do not reorder.
3. Make the default style a config value, as v2 already does (`Routes(mux, defaultStyle)`), so we can A/B test v2 and v3.
4. Watch the metric in the last section before switching the default.

### Face styles: decided against

We built Soft and Bold face families, then a masquerade redesign, and removed both on 2026-10-16. Faces that differ by gender mark women just as the pink names did, and neither design was good enough. Everyone keeps the original society mark; the aura is how people make the face their own.

## Yearning features

Each feature turns a romance-novel beat into something real people do together. None of them fakes interest.

| Feature | Beat it borrows | What it is | Builds on |
| --- | --- | --- | --- |
| Reveal ladder | Slow burn | Face, then chat, then voice note, then photo. Each step needs both people to tap "closer". Either can stay at any step | The knock and door flow in `internal/websocket/hub.go`; `errKnockImage` already holds photos back |
| Remembered conversations | Being noticed | On reconnect, show both people the tags you shared and a line you saved: "you talked about insomnia and cricket" | Past chats already kept in `localStorage` |
| "Pick this up tomorrow?" | The pause between chapters | Ending a chat offers a mutual bookmark. Both see the thread waiting, with a gentle note when the other is online | Doors; needs doors that outlive a session for members |
| Letters | Longing across distance | One long message, delivered hours later. The app *Slowly* built a loyal, female-heavy audience on this | New message kind with a delivery time |
| Conversation prompts | Banter | A card drawn from both people's tags, or a 2am question of the day | Tags on every Card |
| *Her* palette | Warm world | Add Ember and Rose tokens; a private chat shifts to a rose tint once both opened it | `--amber` tokens in `web/style.css` |

The ladder is the most important. The face-first design is already a slow burn; making it explicit and mutual is what makes it feel safe.

### Closeness glow

When two people have been talking for a while, the chat gets warmer and everything else fades back.

| Stage | Trigger (both sides, no message text read) | What changes |
| --- | --- | --- |
| Warm | 15 min and 20+ replies each way | Chat background tints toward rose; the other chats in the sidebar blur slightly |
| Close | 45 min, replies under 2 min apart | A small heart beside their name beats slowly, about 60 bpm; the sidebar dims further |
| Glow | Same pair returns on a second day | The heart stays on their entry in your list; the reveal ladder offers its next step |

Rules:

- Each person sees it only on their own screen, and either can turn it off with one tap ("keep it plain").
- Triggers are counts and timing only, worked out in the client from the messages already there. The server never reads message content for it.
- Respect `prefers-reduced-motion`: no beating, just the tint.
- New messages from other chats still show a badge through the blur, so nobody misses a safety-relevant message.

The glow does not respond to sexual content. Detecting sexting and rewarding it with a heartbeat would mean scanning private messages and pushing strangers, whose age is only self-reported, toward explicit chat. That is the riskiest thing this app could build. Time and back-and-forth already pick out the chats that matter.

## Safety foundations

Ship these before the aura and the ladder. Yearning only works when people feel safe.

| Change | Why | Where |
| --- | --- | --- |
| Gender colour private to strangers | Stops women being spotted and swarmed | `.who-name[data-gender]` rules in `web/style.css`; set `data-gender` only in opened doors |
| Knock-only mode | She decides who reaches her | A setting checked when a knock arrives in `hub.go` |
| Limit new chats per sender | Stops one man sending 40 openers an hour | `internal/ratelimit`, keyed by sender, tighter for new accounts |
| Images blurred until opened | Unwanted images are the top complaint | Client blur with tap to view; `errKnockImage` already blocks images in knocks |
| "Are you sure?" on flagged text | Tinder reported about 10% fewer harmful messages | `internal/filter` already scores text |
| Report shows its outcome | People return when they see reports work | Admin action sends a short notice to the reporter |

## Out of scope

We will not build these. Each one raises retention for a few weeks and then destroys trust, and regulators now name them directly (FTC dark-pattern report 2022, EU Digital Services Act article 25).

- Fake "someone viewed you" or "someone is waiting" notifications
- Bots or staff posing as interested users
- Holding back real replies or matches to stretch sessions
- Visual markers that single out women, by default or by gender
- Sexualised imagery or copy used to pull in men
- Streaks or loss-framed alerts ("your connection will expire")

We also will not build effects triggered by sexual content in messages, for the reasons given under Closeness glow.

## Metrics and guardrails

The north star is women's day-7 return rate. The lever we expect to move it most is unwanted messages in the first session. Rollout order is in the Summary; each step ships behind a config flag and runs as an A/B test.

| Metric | Kind | Goal |
| --- | --- | --- |
| Women returning on day 1 and day 7 | North star | Up |
| Unwanted openers per woman in her first session (ignored knocks plus blocks) | Driver | Down |
| Chats that reach a second reply | Driver | Up |
| Ladder steps reached per chat | Feature health | Up |
| Share of profiles with an aura, by gender | Feature health | Even across genders; a skew toward women means it is acting as a marker |
| Reports and blocks per 100 chats | Guardrail | Must not rise; any test that raises it stops |

Open question: we need a privacy-safe way to log these (IDs and counts only, no message text), per the logging rules in `CLAUDE.md`.
