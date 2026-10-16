# <a id="blob-storage-and-streaming-architecture" >Blob Storage and Streaming Architecture</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Control Plane vs. Data Plane Architectural Separation](#control-plane-vs-data-plane-architectural-separation)
  - [Plain HTTP Media Streaming & Byte-Range Slicing](#plain-http-media-streaming-and-byte-range-slicing)
  - [Content-Addressed SHA-256 Deduplication & Immutable Edge Caching](#content-addressed-sha-256-deduplication-and-immutable-edge-caching)
  - [Multipart & Raw Binary Upload Pipeline](#multipart-and-raw-binary-upload-pipeline)
- [4. Assumptions](#4-assumptions)
- [5. Future work](#5-future-work)
  - [Presigned Object Storage Offloading (S3 / GCS)](#presigned-object-storage-offloading)
  - [Adaptive Bitrate Video Transcoding (HLS)](#adaptive-bitrate-video-transcoding-hls)

---

## <a id="1-background" >1. Background</a>

In Quadsmith, users upload, store, and view large binary assets across three
primary categories:

1. **High-Definition Flight Videos**: FPV flight DVR recordings and build test
   flights (`video/mp4`, `video/webm`, `video/quicktime`) up to 100 MB.
2. **3D CAD Models**: Canopy mounts, arm guards, and camera mounts for 3D
   printing (`model/stl`, `model/3mf`, `model/step`, `model/obj`) up to 35 MB.
3. **High-Resolution Photography & Avatars**: Component hero photos, build
   galleries, and user profile avatars (`image/jpeg`, `image/png`, `image/webp`,
   `image/gif`, `image/svg+xml`, `image/avif`) up to 15 MB.

Binary storage was originally introduced in commit `1993c62` with
content-addressable SHA-256 deduplication in PostgreSQL (`blobs` and
`blob_references` tables), served through plain HTTP handlers in
[`handler.go`](../../src/go/shared/pkg/blob/handler.go) and mounted directly
onto the API server in [`main.go`](../../src/go/server-api/main.go#L137-L144):
`POST /api/blobs` for uploads and `GET /blobs/{id}` for downloads.

During recent architecture reviews, we evaluated whether ConnectRPC streaming
RPCs (e.g.,
`rpc DownloadBlob(DownloadBlobRequest) returns (stream DownloadBlobChunkResponse)`)
could replace these plain HTTP endpoints to achieve 100% protocol consistency
across the Quadsmith API surface.

Our investigation demonstrated that while ConnectRPC streaming RPCs are
technically capable of transmitting chunked binary payloads, they cannot replace
plain HTTP endpoints for media streaming without severely degrading web video
playback, breaking browser caching, and breaking native HTML5 media tags. This
document formalizes these findings into the system architecture.

---

## <a id="2-goal" >2. Goal</a>

Formalize and document the Quadsmith blob storage and media streaming
architecture, establishing an explicit separation between the **ConnectRPC
Control Plane** (metadata, authorization, and entity references) and the **Plain
HTTP Data Plane** (multipart/binary upload ingress and byte-range streaming
egress). Provide a permanent technical record justifying why plain HTTP
endpoints are maintained for binary data transfer over ConnectRPC streaming
RPCs.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Maintain plain HTTP endpoints for all binary data transfers (`/blobs/{id}`,
>   `/api/blobs`).
> - Keep ConnectRPC focused exclusively on metadata, lifecycle, and entity
>   references.
> - Plain HTTP enables native HTML5 `<video>` scrubbing via HTTP 206 Partial
>   Content range requests.
> - Content-addressed SHA-256 paths unlock immutable edge CDN and browser
>   caching.
> - Avoid ConnectRPC wire envelope overhead and iOS Safari Media Source
>   Extensions incompatibility.

Quadsmith adopts an architectural separation of concerns between the API Control
Plane (RPC-driven) and the Media Data Plane (HTTP byte-stream-driven), mirroring
established cloud architectures such as Amazon S3, Google Cloud Storage,
YouTube, and GitHub.

ConnectRPC provides strongly-typed, schema-enforced Protobuf RPCs for metadata,
authorization, and relational references, while high-throughput, low-latency
binary transfers execute directly over standard HTTP/1.1 and HTTP/2 byte
streams.

```mermaid
flowchart TD
    subgraph ControlPlane["Control Plane (ConnectRPC)"]
        UI_Meta["React Frontend / CLI"] -->|ConnectRPC POST| API_Meta["ConnectRPC Services"]
        API_Meta -->|Metadata Queries| DB_Meta[("PostgreSQL (blobs, blob_references)")]
    end

    subgraph DataPlane["Data Plane (Plain HTTP)"]
        UI_Media["HTML5 &lt;video&gt;, &lt;img&gt;, 3D Viewers"] -->|GET /blobs/{id} (Range: bytes=X-Y)| HTTP_Stream["blob.Handler (http.ServeContent)"]
        HTTP_Stream -->|206 Partial Content| UI_Media
        UI_Upload["File Picker / Drag & Drop / CLI"] -->|POST /api/blobs (Multipart/Stream)| HTTP_Upload["blob.Handler (PutBlob)"]
        HTTP_Upload -->|BYTEA Storage| DB_Blob[("PostgreSQL (blobs)")]
        HTTP_Stream -->|Zero-Copy Stream| DB_Blob
    end
```

### <a id="control-plane-vs-data-plane-architectural-separation" >Control Plane vs. Data Plane Architectural Separation</a>

The Quadsmith platform distinguishes strictly between control operations and
data transfer:

1. **Control Plane (ConnectRPC)**:
   - Manages blob metadata, ownership, authorization, and referential integrity.
   - Associates blobs with domain entities (builds, parts, user profiles) via
     the `blob_references` table defined in
     [`src/proto/quadsmith/store/blob.proto`](../../src/proto/quadsmith/store/blob.proto).
   - Future management RPCs (e.g., inspecting quotas, linking/unlinking
     attachments, querying file dimensions or durations) belong in ConnectRPC
     service interfaces.
2. **Data Plane (Plain HTTP)**:
   - Manages raw byte ingress and egress through
     [`blob.Handler`](../../src/go/shared/pkg/blob/handler.go#L20-L45).
   - `POST /api/blobs`: Authenticated streaming upload endpoint accepting both
     `multipart/form-data` and raw octet streams.
   - `GET /blobs/{id}`: Public, content-addressed streaming download endpoint
     supporting HTTP range requests and long-lived client caching.
   - `GET /api/blobs/{id}/metadata`: Lightweight HTTP JSON metadata endpoint for
     direct inspection.

<details>
<summary><a id="alternatives-considered-control-plane-separation" ><strong>Alternatives considered</strong></a></summary>

##### <a id="alt-monolithic-connectrpc-data-plane" >Monolithic ConnectRPC Data Plane for All Transfers</a>

Route all binary uploads and downloads through ConnectRPC streaming RPCs (e.g.,
`stream DownloadBlobChunkResponse` and `stream UploadBlobChunkRequest`) in a
`BlobService` to achieve 100% protocol uniformity across the backend.

**Pros:**

- Unifies 100% of network interactions under ConnectRPC and Protobuf
  definitions.
- Generates fully-typed TypeScript and Go client stubs for media downloads.

**Cons:**

- Incompatible with native browser `<video>`, `<audio>`, and `<img>` tags, which
  cannot parse ConnectRPC wire protocols.
- Adds 5-byte framing and Protobuf field delimiter overhead to every transmitted
  chunk.
- Completely breaks browser disk caching and CDN edge caching for binary media
  assets.

##### <a id="alt-monolithic-rest-api" >Monolithic REST HTTP for All Metadata and Data</a>

Expose traditional REST JSON endpoints for both metadata management and binary
file transfers, eschewing ConnectRPC entirely for blob management.

**Pros:**

- Keeps all blob-related logic within a single protocol paradigm.

**Cons:**

- Conflicts with Quadsmith's architectural standard where all platform business
  logic is defined in Protobuf and exposed via ConnectRPC.
- Loses type-safe Protobuf code generation, schema validation, and reflection
  tools for metadata entities.

</details>

### <a id="plain-http-media-streaming-and-byte-range-slicing" >Plain HTTP Media Streaming & Byte-Range Slicing</a>

High-definition flight video playback imposes strict protocol requirements that
preclude the use of framed RPC protocols:

1. **Native HTML5 Media Decoder Integration**:
   - The browser `<video>` and `<audio>` elements (and native mobile playback
     engines such as Apple AVPlayer and Android ExoPlayer) execute native C++
     network fetches directly in the browser engine.
   - They do not execute application JavaScript, cannot invoke ConnectRPC
     clients, and cannot decode Protobuf envelopes. They require an HTTP URL
     serving a raw media container bitstream (e.g., MP4 or WebM).
2. **HTTP Range Requests & Timeline Scrubbing (`206 Partial Content`)**:
   - When a user scrubs along a video timeline, skips forward 30 seconds, or
     buffers ahead, the browser issues HTTP `GET` requests with
     `Range: bytes=X-Y` headers.
   - In [`handler.go`](../../src/go/shared/pkg/blob/handler.go#L164), Go's
     [`http.ServeContent`](https://pkg.go.dev/net/http#ServeContent) natively
     handles byte-range slicing, conditional `If-Range` checks, and
     `206 Partial Content` status codes from any `io.ReadSeeker`.
   - ConnectRPC streaming RPCs use HTTP `POST` requests under the Connect
     streaming wire format. They do not support HTTP range slicing or standard
     `206 Partial Content` status codes.
3. **Wire Protocol Framing Overhead**:
   - ConnectRPC streams do not transmit raw bytes. Every chunk is enveloped in a
     5-byte frame prefix (1 flag byte + 4 length bytes), serialized into a
     Protobuf message with tag/length varints (`bytes chunk = 1;`), and
     concluded with an end-of-stream trailer frame.
   - Media decoders expecting pure MP4 box headers (`ftyp`, `moov`, `mdat`) fail
     to parse framed RPC streams.
4. **Media Source Extensions (MSE) Limitations**:
   - Consuming ConnectRPC chunks in JavaScript and piping them into an HTML5
     `<video>` element requires the Media Source Extensions (MSE) API.
   - Standard progressive MP4 files uploaded by Quadsmith users cannot be
     appended to an MSE `SourceBuffer` without client-side transmuxing to
     fragmented MP4 (fMP4) via heavy JavaScript libraries (e.g., `mp4box.js`).
   - Apple iOS Safari does not support MSE for standard MP4 video, rendering the
     platform unplayable on iPhones and iPads.
   - Buffering the entire stream in memory as a client-side `Blob`
     (`URL.createObjectURL(blob)`) destroys streaming behavior, forcing users to
     wait for full 100 MB downloads before playback begins and risking tab
     crashes from memory exhaustion.
5. **Kernel Zero-Copy I/O**:
   - Plain HTTP range streaming enables kernel-level zero-copy transfers via
     `sendfile(2)` or `io.Copy`, moving bytes directly from storage to network
     sockets without intermediate user-space buffering or Protobuf serialization
     cycles.

| Media Streaming Capability                | Plain HTTP (`/blobs/{id}`)                              | ConnectRPC Streaming RPC                         |
| :---------------------------------------- | :------------------------------------------------------ | :----------------------------------------------- |
| **Native `<video>` / `<audio>` Playback** | Native (zero JavaScript)                                | Incompatible (fails without custom JS demuxer)   |
| **Timeline Scrubbing / Seeking**          | Native (`Range: bytes=...`, `206 Partial Content`)      | Unsupported (requires manual custom RPC offsets) |
| **Mobile Browser Compatibility**          | 100% (Chrome, Firefox, Safari, iOS, Android)            | Broken on iOS Safari (no MSE for standard MP4)   |
| **Wire Protocol Overhead**                | 0% (pure container bytes)                               | Framing envelope + Protobuf varint delimiters    |
| **Edge CDN & Browser Caching**            | Native (`Cache-Control: immutable`, `304 Not Modified`) | Bypassed by default (RPC POSTs are uncacheable)  |
| **Kernel Zero-Copy Transfer**             | Supported (`sendfile` / `splice`)                       | Unsupported (requires Protobuf message encoding) |
| **3D Asset & Image Loading**              | Native (`<img>`, Three.js `STLLoader`, `3MFLoader`)     | Requires custom fetch, ArrayBuffer conversion    |

<details>
<summary><a id="alternatives-considered-media-streaming" ><strong>Alternatives considered</strong></a></summary>

##### <a id="alt-media-source-extensions-client-streaming" >Client-Side ConnectRPC Stream Consumption via Media Source Extensions</a>

Consume ConnectRPC chunked Protobuf streams in the React frontend, demux MP4
boxes in JavaScript using MP4Box, and feed chunks to the video player using
Media Source Extensions (MSE).

**Pros:**

- Achieves protocol uniformity for media downloads.
- Allows application-level chunk download telemetry and token authentication.

**Cons:**

- Incompatible with iOS Safari, which does not support MSE for standard MP4
  containers.
- Adds heavy JavaScript dependencies and computational overhead for in-browser
  container demuxing.
- Substantially increases video playback startup latency on mobile devices.

##### <a id="alt-in-memory-blob-buffering" >In-Memory Blob Buffering via URL.createObjectURL</a>

Download all ConnectRPC Protobuf chunks into a client-side ArrayBuffer,
construct a JavaScript `Blob`, and attach it to a `<video>` element using
`URL.createObjectURL(blob)`.

**Pros:**

- Avoids complex MSE container transmuxing libraries.

**Cons:**

- Completely eliminates streaming; video playback cannot start until the entire
  100 MB file has finished downloading.
- Causes high risk of mobile browser tab crashes due to excessive RAM
  consumption.
- Disables seeking ahead of the download position.

</details>

### <a id="content-addressed-sha-256-deduplication-and-immutable-edge-caching" >Content-Addressed SHA-256 Deduplication & Immutable Edge Caching</a>

Quadsmith uses content-addressable storage for all binary blobs:

1. **Deduplication**:
   - Every uploaded blob is hashed via SHA-256 in
     [`store.go`](../../src/go/shared/pkg/blob/store.go#L112-L135).
   - If an identical file is uploaded by another user or linked to another
     build, the existing record is reused (`is_duplicate = true`), saving disk
     storage and network bandwidth.
2. **Immutable HTTP Caching**:
   - Because the URL path (`/blobs/<sha256>`) is cryptographically tied to the
     content, blobs are permanently immutable.
   - [`handler.go`](../../src/go/shared/pkg/blob/handler.go#L152-L161) emits
     aggressive caching headers:

     ```http
     Cache-Control: public, max-age=31536000, immutable
     ETag: "<sha256>"
     Accept-Ranges: bytes
     ```

   - Browsers and intermediate edge CDNs (Cloudflare, CloudFront, Fastly) cache
     the response indefinitely. Subsequent page loads or component previews
     retrieve media directly from local disk or edge cache with zero server
     traffic.
   - When clients supply `If-None-Match`, the handler immediately returns
     `304 Not Modified`.
   - In contrast, ConnectRPC streaming calls use HTTP `POST` requests, which
     standard CDNs and browser HTTP caches treat as uncacheable.

<details>
<summary><a id="alternatives-considered-caching" ><strong>Alternatives considered</strong></a></summary>

##### <a id="alt-mutable-entity-scoped-blob-urls" >Mutable Entity-Scoped Blob URLs</a>

Serve media using semantic, entity-bound paths such as
`/api/builds/{uuid}/video` or `/api/users/{uuid}/avatar`.

**Pros:**

- Provides human-readable URLs reflecting entity ownership hierarchy.

**Cons:**

- Precludes the use of `Cache-Control: immutable`, forcing short cache TTLs or
  frequent revalidations.
- Requires active CDN cache purging whenever an entity's media is updated.
- Prevents deduplication when multiple entities share identical assets (e.g.,
  common 3D printable STL files).

</details>

### <a id="multipart-and-raw-binary-upload-pipeline" >Multipart & Raw Binary Upload Pipeline</a>

Uploads are handled through `POST /api/blobs` in
[`handler.go`](../../src/go/shared/pkg/blob/handler.go#L56-L127) with support
for dual ingestion mechanisms:

1. **Multipart Form Uploads (`multipart/form-data`)**:
   - Used by browser file inputs and drag-and-drop file uploaders in the React
     frontend.
   - Streams multipart chunks up to the maximum permitted payload size without
     buffering full request bodies in memory.
2. **Raw Binary Streams (`application/octet-stream`)**:
   - Used by CLI tools ([`qs blob upload`](../../src/go/cli/blob_cmd.go)) and
     script automation.
   - Extracts filenames from query parameters (`?filename=...`) or the
     `X-Filename` HTTP header.
3. **MIME Sniffing & Extension Normalization**:
   - Validates file content using magic byte inspection
     ([`http.DetectContentType`](https://pkg.go.dev/net/http#DetectContentType))
     paired with extension mapping in
     [`NormalizeMIMEType`](../../src/go/shared/pkg/blob/store.go#L74-L100).
   - Enforces strict category size limits: 15 MB for images, 35 MB for 3D
     models, and 100 MB for videos.
   - Rejects unsupported file formats immediately with
     `415 Unsupported Media Type`.

<details>
<summary><a id="alternatives-considered-upload-pipeline" ><strong>Alternatives considered</strong></a></summary>

##### <a id="alt-connectrpc-client-streaming-upload" >ConnectRPC Client-Streaming Upload RPC</a>

Replace `POST /api/blobs` with a client-streaming ConnectRPC method
`UploadBlob(stream UploadBlobChunkRequest) returns (UploadBlobResponse)` where
clients stream Protobuf-wrapped chunks.

**Pros:**

- Unifies upload schemas within Protobuf service definitions.
- Allows structured, typed chunk-level progress metadata.

**Cons:**

- Requires HTTP/2 or HTTP/3 full-duplex streaming in browser
  `fetch(..., { duplex: 'half' })`, which has inconsistent cross-browser support
  and causes buffering failures behind standard reverse proxies.
- Incompatible with standard HTML `<input type="file">` form submissions and
  standard HTTP command-line tools (cURL, Postman).
- Introduces unnecessary Base64 encoding overhead (+33% payload size) if a
  client falls back to JSON-encoded Connect protocol.

</details>

---

## <a id="4-assumptions" >4. Assumptions</a>

1. **Database-Backed Storage**: Storing binary blobs directly in the PostgreSQL
   `blobs` table as `BYTEA` is appropriate for current development and early
   production volumes (up to 100 MB per item).
2. **Standard HTTP Media Protocols**: Target desktop and mobile browsers support
   HTTP/1.1 and HTTP/2 range requests (`Accept-Ranges: bytes`,
   `206 Partial Content`) for HTML5 media elements.
3. **Entity URL References**: All database entities referencing blobs store
   either content-addressed relative paths (`/blobs/<sha256>`) or external URLs
   (such as YouTube embeds), with referential integrity tracked via
   `blob_references`.

---

## <a id="5-future-work" >5. Future work</a>

### <a id="presigned-object-storage-offloading" >Presigned Object Storage Offloading (S3 / GCS)</a>

As media volume expands, offload `BYTEA` payload storage from PostgreSQL to S3
or Google Cloud Storage. Under this design:

- The ConnectRPC control plane will generate presigned upload and download URLs.
- The data plane will remain 100% plain HTTP, transferring bytes directly
  between client browsers and cloud storage buckets without passing media bytes
  through the Quadsmith API server.

### <a id="adaptive-bitrate-video-transcoding-hls" >Adaptive Bitrate Video Transcoding (HLS)</a>

Implement a background video transcoding worker in
[`src/go/server-jobs`](../../src/go/server-jobs) to convert uploaded raw
MP4/WebM videos into HTTP Live Streaming (HLS) multi-rate profiles (`.m3u8`
playlists and `.ts` / `.m4s` segments).

HLS operates entirely over plain HTTP `GET` requests, allowing adaptive quality
switching based on client bandwidth while maintaining native iOS and web video
compatibility.
