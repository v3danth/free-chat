# Free Online Chat India: HTTP and WebSocket API

All bodies are JSON. Every error is `{"error": "<message>"}` with a meaningful
status; internal failures return `500` with a generic message.

Authenticated HTTP calls send `Authorization: Bearer <token>`. Tokens last 24
hours and are revoked instantly by a ban.

---

## 1. Joining

### POST /auth/guest → 201
A one-visit profile card. No email, no password.

```json
{ "name": "राहुल", "gender": "male", "age": 24, "intent": "night_owl",
  "about": "cannot sleep, can talk", "location": "Delhi" }
```

| Field | Rule |
|---|---|
| name | 2–32 letters/digits/underscore in any script, single inner spaces; staff words (admin, moderator, ...) reserved; not unique for guests |
| gender | male, female, non-binary, femboy, other, couple |
| age | 18–99 |
| intent | talk (default), flirt, night_owl, vent, something_real |
| about | up to 140 characters |
| location | optional, up to 40 characters ("Mumbai", "St. John's, NL") |

Text passes the live word filter: a name that trips it is rejected; about and
location are masked, or rejected on a blocked word.

Response: `{ "token": "...", "user": Self }`

### POST /auth/register → 201
Same fields plus `email` and `password` (8–72 bytes). Member names are unique.
Returns `Self`. Members then log in.

### POST /auth/login → 200
`{ "email": "...", "password": "..." }` → `{ "token": "...", "user": Self }`

### GET /me → 200, PATCH /me → 200
PATCH changes only `intent`, `about`, `location` and `photo_id` (an image you
uploaded; `null` clears it). Fields you leave out keep their value. Name, age and gender are fixed for the visit.
Everyone online receives a `presence` update.

### Card and Self
```json
{
  "id": 7, "kind": "guest", "role": "user",
  "name": "राहुल", "gender": "male", "age": 24, "intent": "night_owl",
  "about": "...", "location": "Delhi",
  "country": "IN",
  "photo_blur_url": "/media/blur/<key>.jpg",
  "online_since": 1791490000,
  "avatar_url": "/avatar/v1/<name>",
  "avatar_tier": "rare"
}
```
- `avatar_url` is everyone's pixel face, drawn from their name (section 2b).
- `country` is ISO 3166-1 alpha-2 from the visitor's IP at sign-up (empty
  when unknown). Render it as an SVG flag, not an emoji.
- A Card only carries the blurred photo. `photo_url` (sharp) appears in
  `Self` and in a `door` event: after a knock is answered.

---

## 2. Images

### POST /media/upload → 201
`multipart/form-data` with one field `image` (JPEG, PNG, GIF or WebP, up to
10 MB, at most 40 megapixels). The server re-encodes it, which strips EXIF
and GPS data.

```json
{ "id": 12, "url": "/media/full/<key>.jpg", "thumb_url": "/media/thumb/<key>.jpg",
  "width": 1280, "height": 960 }
```
Errors: 413 too large, 415 not an image, 403 a banned image, 429 more than
20 uploads in 10 minutes.

### GET /media/{full|thumb|blur}/{key}.jpg
Static, immutable (cached for a year). A removed image's files are deleted.

---

## 2b. Faces

Every name has a pixel-art animal face. The same name (ignoring case,
spacing and look-alike characters) always gives the same face, so nothing
is stored. Responses are cached for a year.

### GET /avatar/v1/{name}?scale=1..16&blink=1 → 200 image/png
24 x 24 pixels per scale step (default 1). `blink=1` draws the eyes closed,
for an idle animation. Show it with `image-rendering: pixelated`.

### GET /avatar/v1/{name}/info → 200
```json
{ "name": "night_owl", "tier": "common", "species": "mouse", "animal": "Mouse",
  "colour": "Field", "face": [16, 14], "roundness": 2.19, "ears": "round",
  "ear_size": 3, "eye_shape": "sparkle", "eye_size": 2, "eye_gap": 3,
  "muzzle": [4, 3], "mouth": "o", "markings": "none", "blush": true,
  "extra": "shades", "url": "/avatar/v1/night_owl" }
```
`tier` is common (75%), rare (17%), epic (6%) or legendary (2%).

### GET /avatar/rules → 200
The limits every animal follows (face size, ears, eye shapes and sizes,
mouths, colours), its chance of being drawn, and an `example` name that
draws it. The page at `/faces` is built on these three routes.

---

## 3. WebSocket: GET /ws?token=...

### Server → client

| type | When | Payload |
|---|---|---|
| `hello` | on connect | `you` (Self), `online` (Cards), `rooms` (`[{id, name}]`), `rate_limit` (`{remaining, reset_in_seconds}`) |
| `presence` | someone joins, edits, leaves | `event`: `join` (also a reconnect), `update`, `leave`; `user_id`; `user` (Card, not on leave) |
| `history` | after `join` | `room_id`, `messages`: up to 20 chat events, **newest first** |
| `chat` | a room message | `id, room_id, sender_id, name, content, image?{url, thumb_url}, ts, filtered?` |
| `dm` | a private message (also echoed to the sender) | `id, from, to, content, image?, ts, knock?, filtered?` |
| `door` | a knock was answered | `with`: the other person's Card plus `photo_url` |
| `removed` | a moderator removed a message | `message_id` |
| `notice` | muted, kicked, banned | `code, text, until?` |
| `error` | a command failed | `error`, `code`: INVALID, FORBIDDEN, NOT_FOUND, RATE_LIMITED, ... |

### Client → server

| type | Fields | Notes |
|---|---|---|
| `join` / `leave` | `room_id` | Room 1 is the main room |
| `chat` | `room_id`, `content` and/or `media_id` | Must have joined the room |
| `dm` | `to`, `content` and/or `media_id` | See knocking |
| `block` / `unblock` | `user_id` | Mutual and silent |

### Knocking (private chat)
1. Your first message to someone is a **knock**. It must be text: photos are
   refused until they reply.
2. Until they reply you cannot send a second message (`FORBIDDEN`).
3. When they reply, both sides receive `door` with the other's sharp photo.
4. The door closes when either person leaves. Messaging someone offline, or
   someone on either side of a block, returns the same `NOT_FOUND`.

---

## 4. Safety

### POST /reports → 204
```json
{ "target_type": "message", "target_id": 123, "reason": "harassment", "note": "optional" }
```
`target_type`: message, media, user. `reason`: spam, harassment, nudity,
violence, hate, underage, scam, other. A private message can only be
reported by someone in that conversation. A snapshot of the evidence is kept
with the report. At 3 open reports (configurable) a message or image is
hidden until a moderator reviews it.

---

## 5. Moderation (moderator or admin)

| Method and path | Body | Effect |
|---|---|---|
| GET /admin/reports?status=open | | Report queue, oldest first |
| POST /admin/reports/{id}/dismiss | | |
| POST /admin/messages/{id}/remove | | Hidden everywhere, live |
| POST /admin/media/{id}/remove | | Files deleted, hash banned forever |
| GET /admin/users/{id}/messages | | Their recent messages |
| POST /admin/users/{id}/kick | | Disconnect now |
| POST /admin/users/{id}/mute | `{"minutes": 30}` | |
| POST /admin/users/{id}/ban | `{"minutes": 0, "ip": true, "reason": "..."}` | 0 = permanent; IP bans last at most 7 days |
| POST /admin/users/{id}/unban | | |
| PUT /admin/users/{id}/role | `{"role": "moderator"}` | **Admin only**; members only |
| GET /admin/words | | Banned words |
| POST /admin/words | `{"word": "...", "action": "mask"}` | `mask` or `block`; phrases block only; live at once |
| DELETE /admin/words/{id} | | |
| GET /admin/audit | | Every moderation action, newest first |

Nobody can act on themselves or on someone of equal or higher role.
