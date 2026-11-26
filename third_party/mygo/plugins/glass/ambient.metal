// Single-pass animated ambient gradient backdrop in Metal Shading Language
// (see scene.Effect), matching ambient.hlsl, ambient.glsl and ambient.go.

float3 ambientUnpack(float v) {
	float b = floor(v / 65536.0f);
	float rem = v - b * 65536.0f;
	float g = floor(rem / 256.0f);
	float r = rem - g * 256.0f;
	return float3(r, g, b) / 255.0f;
}

float ambientBez(float t, float a, float b) {
	float u = 1.0f - t;
	return 3.0f * u * u * t * a + 3.0f * u * t * t * b + t * t * t;
}

float ambientEase(float x, int mode) {
	if (mode == 0 || x <= 0.0f || x >= 1.0f) {
		return x;
	}
	float x1 = mode == 2 ? 0.42f : 0.25f;
	float y1 = mode == 2 ? 0.00f : 0.10f;
	float x2 = mode == 2 ? 0.58f : 0.25f;
	float y2 = 1.0f;
	float lo = 0.0f;
	float hi = 1.0f;
	for (int k = 0; k < 18; k++) {
		float mid = 0.5f * (lo + hi);
		if (ambientBez(mid, x1, x2) < x) {
			lo = mid;
		} else {
			hi = mid;
		}
	}
	return ambientBez(0.5f * (lo + hi), y1, y2);
}

float ambientIGN(float2 pos) {
	float n = fract(52.9829189f * fract(dot(pos, float2(0.06711056f, 0.00583715f))));
	float r = n * 2.0f - 1.0f;
	return (r >= 0.0f ? 1.0f : -1.0f) * (1.0f - sqrt(max(0.0f, 1.0f - abs(r))));
}

float3 ambientHardLight(float3 a, float3 b) {
	float3 lo = 2.0f * a * b;
	float3 hi = 1.0f - 2.0f * (1.0f - a) * (1.0f - b);
	return float3(
		b.r < 0.5f ? lo.r : hi.r,
		b.g < 0.5f ? lo.g : hi.g,
		b.b < 0.5f ? lo.b : hi.b
	);
}

float4 effect(float2 p, Effect e, texture2d<float> backdrop) {
	(void)backdrop;
	float w = e.rect.z;
	float h = e.rect.w;
	if (w <= 0.0f || h <= 0.0f) {
		return float4(0.0f);
	}
	float2 q = p - e.rect.xy;

	float3 baseFrom = ambientUnpack(e.p0.x);
	float3 baseTo = ambientUnpack(e.p0.y);
	float3 colors[5] = {
		ambientUnpack(e.p0.z),
		ambientUnpack(e.p0.w),
		ambientUnpack(e.p1.x),
		ambientUnpack(e.p1.y),
		ambientUnpack(e.p1.z),
	};

	int variant = int(floor(e.p1.w + 0.5f));
	int count = clamp(int(floor(e.p2.x + 0.5f)), 1, 5);
	float seconds = max(0.0f, e.p2.y);
	float keep = e.p2.z > 0.5f ? (1.0f - 128.0f / 255.0f) : (1.0f - 61.0f / 255.0f);
	float opacity = e.p2.w <= 0.0f ? 1.0f : min(e.p2.w, 1.0f);
	float scale = e.p3.x <= 0.0f ? 1.0f : e.p3.x;

	constant float sin40 = 0.6427876096865393f;
	constant float cos40 = 0.766044443118978f;
	float gradLen = max(abs(w * sin40) + abs(h * (-cos40)), 1.0f);
	float gradT = saturate(0.5f + ((q.x - 0.5f * w) * sin40 + (q.y - 0.5f * h) * (-cos40)) / gradLen);
	float3 col = mix(baseFrom, baseTo, gradT);

	constant float orbsLeft[5] = {0.05f, -0.02f, 0.22f, 0.15f, 0.18f};
	constant float orbsTop[5] = {0.0f, 0.15f, 0.08f, 0.22f, -0.05f};
	constant float orbsOx[5] = {0.0f, -400.0f, 400.0f, -200.0f, -800.0f};
	constant float orbsOy[5] = {0.0f, 0.0f, 0.0f, 0.0f, -800.0f};
	constant float orbsDur[5] = {30.0f, 20.0f, 40.0f, 40.0f, 20.0f};
	constant int orbsRev[5] = {0, 1, 0, 0, 0};
	constant int orbsEase[5] = {1, 1, 0, 1, 1};
	constant int orbsMotion[5] = {1, 0, 0, 2, 0};

	constant float auroraTop[5] = {0.2f, 0.37f, 0.12f, 0.48f, 0.3f};
	constant float auroraDur[5] = {36.0f, 44.0f, 30.0f, 48.0f, 40.0f};
	constant int auroraRev[5] = {0, 1, 0, 1, 0};

	constant float chaosLeft[5] = {0.05f, 0.3f, 0.12f, 0.25f, 0.04f};
	constant float chaosTop[5] = {0.0f, 0.08f, 0.32f, 0.0f, 0.26f};
	constant float chaosDur[5] = {14.0f, 18.0f, 11.0f, 16.0f, 13.0f};
	constant int chaosRev[5] = {0, 1, 0, 1, 0};

	constant float raysDur[5] = {30.0f, 40.0f, 24.0f, 36.0f, 28.0f};
	constant int raysRev[5] = {0, 1, 0, 1, 0};
	constant float raysStart[5] = {0.0f, 90.0f, 180.0f, 270.0f, 45.0f};
	constant float raysSpan[5] = {80.0f, 70.0f, 100.0f, 60.0f, 90.0f};

	constant float3 chaosPts[5] = {
		float3(-0.3f, -0.35f, 1.0f),
		float3(0.35f, 0.05f, 1.3f),
		float3(-0.15f, 0.4f, 0.75f),
		float3(0.25f, -0.3f, 1.15f),
		float3(-0.3f, -0.35f, 1.0f),
	};

	constant float PI = 3.141592653589793f;

	for (int i = 0; i < count; i++) {
		float bw, bh, left, top, ox, oy, blobOpacity, stop, blurPx, dur;
		int rev, easeMode, motion;
		bool isEllipse = false;
		bool isConic = false;
		float conicStart = 0.0f;
		float conicSpan = 0.0f;

		if (variant == 1) {
			bw = 1.4f * w;
			bh = 0.36f * h;
			left = -0.2f * w;
			top = auroraTop[i] * h;
			ox = 0.5f * bw;
			oy = 0.5f * bh;
			blobOpacity = 0.55f;
			stop = 0.55f;
			isEllipse = true;
			blurPx = 44.0f * scale;
			dur = auroraDur[i];
			rev = auroraRev[i];
			easeMode = 2;
			motion = 3;
		} else if (variant == 2) {
			bw = 0.6f * w;
			bh = 0.6f * h;
			left = chaosLeft[i] * w;
			top = chaosTop[i] * h;
			ox = 0.5f * bw;
			oy = 0.5f * bh;
			blobOpacity = 0.75f;
			stop = 0.55f;
			blurPx = 30.0f * scale;
			dur = chaosDur[i];
			rev = chaosRev[i];
			easeMode = 2;
			motion = 4;
		} else if (variant == 3) {
			bw = 1.5f * w;
			bh = 1.5f * h;
			left = (i == 0 ? 0.05f : -0.25f) * w;
			top = (i == 0 ? 0.0f : -0.25f) * h;
			ox = 0.5f * bw;
			oy = 0.5f * bh;
			blobOpacity = 0.45f;
			stop = 0.5f;
			isConic = true;
			blurPx = 18.0f * scale;
			conicStart = raysStart[i];
			conicSpan = raysSpan[i];
			dur = raysDur[i];
			rev = raysRev[i];
			easeMode = 0;
			motion = 0;
		} else {
			bw = 0.8f * w;
			bh = 0.8f * h;
			left = orbsLeft[i] * w;
			top = orbsTop[i] * h;
			ox = 0.5f * bw + orbsOx[i] * scale;
			oy = 0.5f * bh + orbsOy[i] * scale;
			blobOpacity = (i == 3) ? 0.7f : 0.8f;
			stop = 0.5f;
			blurPx = 64.0f * scale;
			dur = orbsDur[i];
			rev = orbsRev[i];
			easeMode = orbsEase[i];
			motion = orbsMotion[i];
		}

		float progress = fract(seconds / dur);
		if (rev != 0) {
			progress = 1.0f - progress;
		}
		int segments = (motion == 4) ? 4 : 2;
		float segFloat = progress * float(segments);
		int segment = min(segments - 1, max(0, int(floor(segFloat))));
		float local = ambientEase(segFloat - float(segment), easeMode);

		float tx = 0.0f;
		float ty = 0.0f;
		float rot = 0.0f;
		float sc = 1.0f;

		if (motion == 0) {
			rot = (float(segment) + local) * PI;
		} else if (motion == 1) {
			ty = (segment == 1 ? (0.5f - local) : (-0.5f + local)) * bh;
		} else if (motion == 2) {
			float k = -1.0f + 2.0f * local;
			if (segment == 1) {
				k = -k;
			}
			tx = k * 0.5f * bw;
			ty = k * 0.1f * bh;
		} else if (motion == 3) {
			float k = -1.0f + 2.0f * local;
			if (segment == 1) {
				k = -k;
			}
			tx = k * 0.45f * bw;
			ty = k * 0.06f * bh;
			rot = k * 3.0f * PI / 180.0f;
		} else {
			float3 aPt = chaosPts[segment];
			float3 zPt = chaosPts[segment + 1];
			tx = mix(aPt.x, zPt.x, local) * bw;
			ty = mix(aPt.y, zPt.y, local) * bh;
			sc = mix(aPt.z, zPt.z, local);
		}

		float2 d = q - float2(left + tx + ox, top + ty + oy);
		float co = cos(rot);
		float si = sin(rot);
		float lx = (co * d.x + si * d.y) / sc + ox;
		float ly = (-si * d.x + co * d.y) / sc + oy;
		float2 bpos = float2(lx - 0.5f * bw, ly - 0.5f * bh);

		float alpha = 0.0f;
		if (isConic) {
			float halfSpan = 0.5f * conicSpan;
			float deg = (abs(bpos.x) + abs(bpos.y) > 1e-5f) ? atan2(bpos.x, -bpos.y) * (180.0f / PI) : 0.0f;
			float ang = deg - conicStart - halfSpan;
			ang -= 360.0f * floor((ang + 180.0f) / 360.0f);
			float rPx = length(bpos);
			float blurDeg = atan2(blurPx, max(rPx, 1.0f)) * (180.0f / PI);
			float angAlpha = 1.0f - smoothstep(0.0f, 1.0f, saturate(abs(ang) / (halfSpan + 1.5f * blurDeg)));
			float rNorm = length(float2(bpos.x / max(0.5f * bw, 1.0f), bpos.y / max(0.5f * bh, 1.0f)));
			float radFade = 1.0f - smoothstep(0.55f, 1.0f, rNorm);
			alpha = angAlpha * radFade;
		} else if (isEllipse) {
			float dist = length(float2(bpos.x / max(0.5f * bw, 1.0f), bpos.y / max(0.5f * bh, 1.0f))) / 1.41421356f;
			float blurNorm = min(1.5f * blurPx / max(0.5f * min(bw, bh), 1.0f), 0.45f);
			float u = dist / (stop + blurNorm);
			alpha = 1.0f - smoothstep(0.0f, 1.0f, saturate(u));
		} else {
			float halfDiag = max(sqrt(0.25f * bw * bw + 0.25f * bh * bh), 1.0f);
			float dist = length(bpos) / halfDiag;
			float blurNorm = min(1.5f * blurPx / halfDiag, 0.45f);
			float u = dist / (stop + blurNorm);
			alpha = 1.0f - smoothstep(0.0f, 1.0f, saturate(u));
		}
		alpha *= blobOpacity;
		if (alpha > 0.0f) {
			col = mix(col, ambientHardLight(col, colors[i]), alpha);
		}
	}

	float dither = ambientIGN(floor(p)) * (0.5f / 255.0f);
	col = saturate(col * keep + dither);
	return float4(col * opacity, opacity);
}
