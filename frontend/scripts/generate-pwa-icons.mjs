// Generates the PWA raster icons (192px + 512px PNG) from the same geometry as
// public/icon.svg, so the manifest offers raster icons for installability on
// platforms that prefer PNG (notably iOS apple-touch-icon).
//
// Dependency-free on purpose: the project has no SVG rasteriser available, so we
// draw the icon's analytic shapes directly with 4x supersampling for crisp
// edges and encode a PNG using Node's built-in zlib. Re-run with:
//   npm run icons
//
// The geometry mirrors icon.svg (512x512 viewBox):
//   - rounded-rect background, radius 96, fill #1565C0
//   - centre figure: head circle + body (rounded-top block), white
//   - two teammate circles (r40) at the sides, white @ 0.85 opacity
import { writeFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { deflateSync } from 'zlib';
import { Buffer } from 'buffer';

const OUT_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'public');
const SS = 4; // supersampling factor for anti-aliasing

const BG = [0x15, 0x65, 0xc0];
const WHITE = [0xff, 0xff, 0xff];

function inCircle(x, y, cx, cy, r) {
  const dx = x - cx;
  const dy = y - cy;
  return dx * dx + dy * dy <= r * r;
}

// Point inside an axis-aligned rounded rectangle { x0, y0, x1, y1, r }.
function inRoundedRect(x, y, { x0, y0, x1, y1, r }) {
  if (x < x0 || x > x1 || y < y0 || y > y1) return false;
  const ix0 = x0 + r;
  const ix1 = x1 - r;
  const iy0 = y0 + r;
  const iy1 = y1 - r;
  const cx = x < ix0 ? ix0 : x > ix1 ? ix1 : x;
  const cy = y < iy0 ? iy0 : y > iy1 ? iy1 : y;
  const dx = x - cx;
  const dy = y - cy;
  return dx * dx + dy * dy <= r * r;
}

// Returns the icon colour at a coordinate in the 512-unit space, or null (outside
// the rounded background, i.e. transparent).
function sampleColor(x, y) {
  if (!inRoundedRect(x, y, { x0: 0, y0: 0, x1: 512, y1: 512, r: 96 })) return null;

  // Teammate circles sit behind the centre figure, slightly translucent.
  const teammate = inCircle(x, y, 152, 208, 40) || inCircle(x, y, 360, 208, 40);

  // Centre figure: head + body (rounded-top block down to y=400).
  const head = inCircle(x, y, 256, 182, 62);
  const body = inRoundedRect(x, y, { x0: 152, y0: 298, x1: 360, y1: 400, r: 80 }) && y <= 400;
  const figure = head || body;

  if (figure) return WHITE;
  if (teammate) {
    // 0.85 white over the blue background.
    return BG.map((c) => Math.round(c * 0.15 + 255 * 0.85));
  }
  return BG;
}

function renderRGBA(size) {
  const buf = Buffer.alloc(size * size * 4);
  const scale = 512 / size;
  for (let py = 0; py < size; py++) {
    for (let px = 0; px < size; px++) {
      let r = 0;
      let g = 0;
      let b = 0;
      let a = 0;
      for (let sy = 0; sy < SS; sy++) {
        for (let sx = 0; sx < SS; sx++) {
          const x = (px + (sx + 0.5) / SS) * scale;
          const y = (py + (sy + 0.5) / SS) * scale;
          const c = sampleColor(x, y);
          if (c) {
            r += c[0];
            g += c[1];
            b += c[2];
            a += 255;
          }
        }
      }
      const n = SS * SS;
      const o = (py * size + px) * 4;
      // Average colour over covered subpixels; alpha reflects coverage.
      const cov = a / n;
      buf[o] = cov ? Math.round(r / (a / 255)) : 0;
      buf[o + 1] = cov ? Math.round(g / (a / 255)) : 0;
      buf[o + 2] = cov ? Math.round(b / (a / 255)) : 0;
      buf[o + 3] = Math.round(cov);
    }
  }
  return buf;
}

// --- Minimal PNG encoder (truecolour + alpha, no interlace) ----------------
function crc32(buf) {
  let c = ~0;
  for (let i = 0; i < buf.length; i++) {
    c ^= buf[i];
    for (let k = 0; k < 8; k++) c = c & 1 ? (c >>> 1) ^ 0xedb88320 : c >>> 1;
  }
  return ~c >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length, 0);
  const typeBuf = Buffer.from(type, 'ascii');
  const body = Buffer.concat([typeBuf, data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body), 0);
  return Buffer.concat([len, body, crc]);
}

function encodePng(rgba, size) {
  const sig = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // colour type RGBA
  // compression/filter/interlace default 0
  const stride = size * 4;
  const raw = Buffer.alloc((stride + 1) * size);
  for (let y = 0; y < size; y++) {
    raw[y * (stride + 1)] = 0; // filter: none
    rgba.copy(raw, y * (stride + 1) + 1, y * stride, y * stride + stride);
  }
  const idat = deflateSync(raw, { level: 9 });
  return Buffer.concat([sig, chunk('IHDR', ihdr), chunk('IDAT', idat), chunk('IEND', Buffer.alloc(0))]);
}

for (const size of [192, 512]) {
  const png = encodePng(renderRGBA(size), size);
  const file = join(OUT_DIR, `icon-${size}.png`);
  writeFileSync(file, png);
  // eslint-disable-next-line no-console
  console.log(`wrote ${file} (${png.length} bytes)`);
}
