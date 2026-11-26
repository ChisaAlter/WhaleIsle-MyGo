// Single-pass animated ambient gradient backdrop in GLSL (see scene.Effect),
// compatible with GLSL 3.30 and GLSL ES 3.00, matching ambient.hlsl,
// ambient.metal and ambient.go.

vec3 ambientUnpack(float v) {
	float b = floor(v / 65536.0);
	float rem = v - b * 65536.0;
	float g = floor(rem / 256.0);
	float r = rem - g * 256.0;
	return vec3(r, g, b) / 255.0;
}

float ambientBez(float t, float a, float b) {
	float u = 1.0 - t;
	return 3.0 * u * u * t * a + 3.0 * u * t * t * b + t * t * t;
}

float ambientEase(float x, int mode) {
	if (mode == 0 || x <= 0.0 || x >= 1.0) {
		return x;
	}
	float x1 = mode == 2 ? 0.42 : 0.25;
	float y1 = mode == 2 ? 0.00 : 0.10;
	float x2 = mode == 2 ? 0.58 : 0.25;
	float y2 = 1.0;
	float lo = 0.0;
	float hi = 1.0;
	for (int k = 0; k < 18; k++) {
		float mid = 0.5 * (lo + hi);
		if (ambientBez(mid, x1, x2) < x) {
			lo = mid;
		} else {
			hi = mid;
		}
	}
	return ambientBez(0.5 * (lo + hi), y1, y2);
}

float ambientIGN(vec2 pos) {
	float n = fract(52.9829189 * fract(dot(pos, vec2(0.06711056, 0.00583715))));
	float r = n * 2.0 - 1.0;
	return (r >= 0.0 ? 1.0 : -1.0) * (1.0 - sqrt(max(0.0, 1.0 - abs(r))));
}

vec3 ambientHardLight(vec3 a, vec3 b) {
	vec3 lo = 2.0 * a * b;
	vec3 hi = 1.0 - 2.0 * (1.0 - a) * (1.0 - b);
	return vec3(
		b.r < 0.5 ? lo.r : hi.r,
		b.g < 0.5 ? lo.g : hi.g,
		b.b < 0.5 ? lo.b : hi.b
	);
}

vec4 effect(vec2 p, Effect e) {
	float w = e.rect.z;
	float h = e.rect.w;
	if (w <= 0.0 || h <= 0.0) {
		return vec4(0.0);
	}
	vec2 q = p - e.rect.xy;

	vec3 baseFrom = ambientUnpack(e.p0.x);
	vec3 baseTo = ambientUnpack(e.p0.y);
	vec3 colors[5];
	colors[0] = ambientUnpack(e.p0.z);
	colors[1] = ambientUnpack(e.p0.w);
	colors[2] = ambientUnpack(e.p1.x);
	colors[3] = ambientUnpack(e.p1.y);
	colors[4] = ambientUnpack(e.p1.z);

	int variant = int(floor(e.p1.w + 0.5));
	int count = clamp(int(floor(e.p2.x + 0.5)), 1, 5);
	float seconds = max(0.0, e.p2.y);
	float keep = e.p2.z > 0.5 ? (1.0 - 128.0 / 255.0) : (1.0 - 61.0 / 255.0);
	float opacity = e.p2.w <= 0.0 ? 1.0 : min(e.p2.w, 1.0);
	float scale = e.p3.x <= 0.0 ? 1.0 : e.p3.x;

	const float sin40 = 0.6427876096865393;
	const float cos40 = 0.766044443118978;
	float gradLen = max(abs(w * sin40) + abs(h * (-cos40)), 1.0);
	float gradT = clamp(0.5 + ((q.x - 0.5 * w) * sin40 + (q.y - 0.5 * h) * (-cos40)) / gradLen, 0.0, 1.0);
	vec3 col = mix(baseFrom, baseTo, gradT);

	float orbsLeft[5] = float[5](0.05, -0.02, 0.22, 0.15, 0.18);
	float orbsTop[5] = float[5](0.0, 0.15, 0.08, 0.22, -0.05);
	float orbsOx[5] = float[5](0.0, -400.0, 400.0, -200.0, -800.0);
	float orbsOy[5] = float[5](0.0, 0.0, 0.0, 0.0, -800.0);
	float orbsDur[5] = float[5](30.0, 20.0, 40.0, 40.0, 20.0);
	int orbsRev[5] = int[5](0, 1, 0, 0, 0);
	int orbsEase[5] = int[5](1, 1, 0, 1, 1);
	int orbsMotion[5] = int[5](1, 0, 0, 2, 0);

	float auroraTop[5] = float[5](0.2, 0.37, 0.12, 0.48, 0.3);
	float auroraDur[5] = float[5](36.0, 44.0, 30.0, 48.0, 40.0);
	int auroraRev[5] = int[5](0, 1, 0, 1, 0);

	float chaosLeft[5] = float[5](0.05, 0.3, 0.12, 0.25, 0.04);
	float chaosTop[5] = float[5](0.0, 0.08, 0.32, 0.0, 0.26);
	float chaosDur[5] = float[5](14.0, 18.0, 11.0, 16.0, 13.0);
	int chaosRev[5] = int[5](0, 1, 0, 1, 0);

	float raysDur[5] = float[5](30.0, 40.0, 24.0, 36.0, 28.0);
	int raysRev[5] = int[5](0, 1, 0, 1, 0);
	float raysStart[5] = float[5](0.0, 90.0, 180.0, 270.0, 45.0);
	float raysSpan[5] = float[5](80.0, 70.0, 100.0, 60.0, 90.0);

	vec3 chaosPts[5] = vec3[5](
		vec3(-0.3, -0.35, 1.0),
		vec3(0.35, 0.05, 1.3),
		vec3(-0.15, 0.4, 0.75),
		vec3(0.25, -0.3, 1.15),
		vec3(-0.3, -0.35, 1.0)
	);

	const float PI = 3.141592653589793;

	for (int i = 0; i < 5; i++) {
		if (i >= count) {
			break;
		}
		float bw, bh, left, top, ox, oy, blobOpacity, stop, blurPx, dur;
		int rev, easeMode, motion;
		bool isEllipse = false;
		bool isConic = false;
		float conicStart = 0.0;
		float conicSpan = 0.0;

		if (variant == 1) {
			bw = 1.4 * w;
			bh = 0.36 * h;
			left = -0.2 * w;
			top = auroraTop[i] * h;
			ox = 0.5 * bw;
			oy = 0.5 * bh;
			blobOpacity = 0.55;
			stop = 0.55;
			isEllipse = true;
			blurPx = 44.0 * scale;
			dur = auroraDur[i];
			rev = auroraRev[i];
			easeMode = 2;
			motion = 3;
		} else if (variant == 2) {
			bw = 0.6 * w;
			bh = 0.6 * h;
			left = chaosLeft[i] * w;
			top = chaosTop[i] * h;
			ox = 0.5 * bw;
			oy = 0.5 * bh;
			blobOpacity = 0.75;
			stop = 0.55;
			blurPx = 30.0 * scale;
			dur = chaosDur[i];
			rev = chaosRev[i];
			easeMode = 2;
			motion = 4;
		} else if (variant == 3) {
			bw = 1.5 * w;
			bh = 1.5 * h;
			left = (i == 0 ? 0.05 : -0.25) * w;
			top = (i == 0 ? 0.0 : -0.25) * h;
			ox = 0.5 * bw;
			oy = 0.5 * bh;
			blobOpacity = 0.45;
			stop = 0.5;
			isConic = true;
			blurPx = 18.0 * scale;
			conicStart = raysStart[i];
			conicSpan = raysSpan[i];
			dur = raysDur[i];
			rev = raysRev[i];
			easeMode = 0;
			motion = 0;
		} else {
			bw = 0.8 * w;
			bh = 0.8 * h;
			left = orbsLeft[i] * w;
			top = orbsTop[i] * h;
			ox = 0.5 * bw + orbsOx[i] * scale;
			oy = 0.5 * bh + orbsOy[i] * scale;
			blobOpacity = (i == 3) ? 0.7 : 0.8;
			stop = 0.5;
			blurPx = 64.0 * scale;
			dur = orbsDur[i];
			rev = orbsRev[i];
			easeMode = orbsEase[i];
			motion = orbsMotion[i];
		}

		float progress = fract(seconds / dur);
		if (rev != 0) {
			progress = 1.0 - progress;
		}
		int segments = (motion == 4) ? 4 : 2;
		float segFloat = progress * float(segments);
		int segment = min(segments - 1, max(0, int(floor(segFloat))));
		float local = ambientEase(segFloat - float(segment), easeMode);

		float tx = 0.0;
		float ty = 0.0;
		float rot = 0.0;
		float sc = 1.0;

		if (motion == 0) {
			rot = (float(segment) + local) * PI;
		} else if (motion == 1) {
			ty = (segment == 1 ? (0.5 - local) : (-0.5 + local)) * bh;
		} else if (motion == 2) {
			float k = -1.0 + 2.0 * local;
			if (segment == 1) {
				k = -k;
			}
			tx = k * 0.5 * bw;
			ty = k * 0.1 * bh;
		} else if (motion == 3) {
			float k = -1.0 + 2.0 * local;
			if (segment == 1) {
				k = -k;
			}
			tx = k * 0.45 * bw;
			ty = k * 0.06 * bh;
			rot = k * 3.0 * PI / 180.0;
		} else {
			vec3 aPt = chaosPts[segment];
			vec3 zPt = chaosPts[segment + 1];
			tx = mix(aPt.x, zPt.x, local) * bw;
			ty = mix(aPt.y, zPt.y, local) * bh;
			sc = mix(aPt.z, zPt.z, local);
		}

		vec2 d = q - vec2(left + tx + ox, top + ty + oy);
		float co = cos(rot);
		float si = sin(rot);
		float lx = (co * d.x + si * d.y) / sc + ox;
		float ly = (-si * d.x + co * d.y) / sc + oy;
		vec2 bpos = vec2(lx - 0.5 * bw, ly - 0.5 * bh);

		float alpha = 0.0;
		if (isConic) {
			float halfSpan = 0.5 * conicSpan;
			float deg = (abs(bpos.x) + abs(bpos.y) > 1e-5) ? atan(bpos.x, -bpos.y) * (180.0 / PI) : 0.0;
			float ang = deg - conicStart - halfSpan;
			ang -= 360.0 * floor((ang + 180.0) / 360.0);
			float rPx = length(bpos);
			float blurDeg = atan(blurPx, max(rPx, 1.0)) * (180.0 / PI);
			float angAlpha = 1.0 - smoothstep(0.0, 1.0, clamp(abs(ang) / (halfSpan + 1.5 * blurDeg), 0.0, 1.0));
			float rNorm = length(vec2(bpos.x / max(0.5 * bw, 1.0), bpos.y / max(0.5 * bh, 1.0)));
			float radFade = 1.0 - smoothstep(0.55, 1.0, rNorm);
			alpha = angAlpha * radFade;
		} else if (isEllipse) {
			float dist = length(vec2(bpos.x / max(0.5 * bw, 1.0), bpos.y / max(0.5 * bh, 1.0))) / 1.41421356;
			float blurNorm = min(1.5 * blurPx / max(0.5 * min(bw, bh), 1.0), 0.45);
			float u = dist / (stop + blurNorm);
			alpha = 1.0 - smoothstep(0.0, 1.0, clamp(u, 0.0, 1.0));
		} else {
			float halfDiag = max(sqrt(0.25 * bw * bw + 0.25 * bh * bh), 1.0);
			float dist = length(bpos) / halfDiag;
			float blurNorm = min(1.5 * blurPx / halfDiag, 0.45);
			float u = dist / (stop + blurNorm);
			alpha = 1.0 - smoothstep(0.0, 1.0, clamp(u, 0.0, 1.0));
		}
		alpha *= blobOpacity;
		if (alpha > 0.0) {
			col = mix(col, ambientHardLight(col, colors[i]), alpha);
		}
	}

	float dither = ambientIGN(floor(p)) * (0.5 / 255.0);
	col = clamp(col * keep + dither, 0.0, 1.0);
	return vec4(col * opacity, opacity);
}
