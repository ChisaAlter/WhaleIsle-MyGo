# Third-Party Licenses

This document identifies the third-party software included in the independently authored WhaleIsle-MyGo OpenUI viewer browser runtime and reproduces its complete published license notices. These terms apply to the named dependencies; they do not assign a license to independently authored application files.

## Scope and reproducibility

- Inventory date: 2026-10-17 UTC.
- Runtime: independently authored component library using the official OpenUI language SDK.
- The production browser bundle contains code from **83 package paths**. All have complete published license files.
- Package metadata labels: 79 MIT, 2 ISC, 1 BSD-3-Clause and 1 BSD-2-Clause. Lucide also includes Feather-derived MIT terms, reproduced in full below.
- Package selection comes from positive `bytesInOutput` entries in `dist/bundle-metafile.json`, cross-checked with `dist/used-package-paths.json`.
- Exact installed versions and package tarball integrity values are recorded in `package-lock.json`.
- `package-lock.json` SHA-256: `7d46de5a7add6713ea32aec1d67dff45c0348b3fb14286897300812d7478b048`.
- Build-only tools and packages that contribute no browser runtime bytes are outside this browser inventory. The generated `dist/catalogue-tool.cjs` is a build helper and is not part of the viewer redistribution.
- These license records are static dependency/asset evidence, not a claim of runtime or compatibility testing.

## Official OpenUI SDK provenance

The exact source commits below are recorded in the packages’ published npm SLSA provenance records. This records the registry-provided provenance; it does not claim independent cryptographic signature verification. Each installed package includes the same full MIT LICENSE, copyright 2011–2024 Thesys Inc. The license is reproduced under Notice 1 and also retained in `licenses/OpenUI-MIT.txt`.

| Package | Version | Source commit | Official license | Published provenance |
| --- | --- | --- | --- | --- |
| `@openuidev/lang-core` | `1.0.0` | [`f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20`](https://github.com/thesysdev/openui/tree/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20) | [MIT LICENSE](https://github.com/thesysdev/openui/blob/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20/LICENSE) | [npm record](https://registry.npmjs.org/-/npm/v1/attestations/@openuidev%2flang-core@1.0.0) |
| `@openuidev/react-lang` | `1.0.0` | [`f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20`](https://github.com/thesysdev/openui/tree/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20) | [MIT LICENSE](https://github.com/thesysdev/openui/blob/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20/LICENSE) | [npm record](https://registry.npmjs.org/-/npm/v1/attestations/@openuidev%2freact-lang@1.0.0) |
| `@openuidev/observability` | `0.0.4` | [`b30b1073882f289569be5805be99292203fce60d`](https://github.com/thesysdev/openui/tree/b30b1073882f289569be5805be99292203fce60d) | [MIT LICENSE](https://github.com/thesysdev/openui/blob/b30b1073882f289569be5805be99292203fce60d/LICENSE) | [npm record](https://registry.npmjs.org/-/npm/v1/attestations/@openuidev%2fobservability@0.0.4) |

The SHA-256 of each of the three published SDK LICENSE files is `b6bdd2d3d161722fb3f3afc039114cfa9f3236f036e1c98e0853bc3a85f6cd9e`. The upstream root LICENSE at both recorded commits matches these installed license bytes.

## Local syntax-highlighting dependency provenance

The Code and Markdown syntax highlighter adds the following official npm packages. Their registry tarball URLs and integrity values were matched to the lockfile, and their installed complete license files were byte-for-byte matched to the corresponding official version-tag license files. The exact `gitHead` values below are those published in npm version metadata.

| Package | Version | Terms | npm gitHead | Official version license | License-file SHA-256 |
| --- | --- | --- | --- | --- | --- |
| `lowlight` | `3.3.0` | MIT | `0f36148072cd096ca86753d6f1ff01589d30d78f` | [Version license](https://github.com/wooorm/lowlight/blob/3.3.0/license) | `dd1081884a92952802f4803110a6bb543acea9a814c786d58605b4c1219b5ebb` |
| `highlight.js` | `11.11.1` | BSD-3-Clause | `08cb242e7d4aee787114eb04cc7ab18314d82f92` | [Version license](https://github.com/highlightjs/highlight.js/blob/11.11.1/LICENSE) | `6c081431591d9df696c82dc598fe1423765b8a299b200ed00b281afd0f64c490` |

`highlight.js@11.11.1` satisfies `lowlight@3.3.0`'s declared `~11.11.0` dependency range. The full BSD-3-Clause notice is reproduced below, including source and binary redistribution conditions, the non-endorsement condition, and the warranty/liability disclaimer. It is not an MIT-only dependency. `devlop` contributes no bytes to the final browser bundle and is not part of this runtime inventory.

## Static HTML parser dependency provenance

HtmlView static script preparation uses `parse5@8.0.0` (MIT) and its `entities@6.0.1` dependency (BSD-2-Clause). These exact official npm versions are recorded with tarball URLs and integrity values in the lockfile. Both installed packages contain complete published LICENSE files; their bytes and hashes are reproduced below and included in the generated runtime notices/provenance. No original application parser or component implementation is included. Official package sources: [parse5](https://github.com/inikulin/parse5), [entities](https://github.com/fb55/entities).

## Asset-specific terms

- **KaTeX 0.16.47:** this viewer uses the JavaScript renderer with MathML output. No KaTeX stylesheet or font binaries occur in the production bundle inputs. Its included code uses the published MIT license reproduced below. The separate SIL OFL terms for KaTeX font binaries are therefore not part of this distribution. If a future build embeds those fonts, include their actual copyright/reserved-font-name declarations and full OFL terms.
- **Lucide React 0.575.0:** preserve both the ISC notice and the MIT notice for portions derived from Feather. The full combined published LICENSE appears below.
- The self-contained `dist/viewer.html` embeds the complete runtime notices. `dist/THIRD-PARTY-NOTICES.txt` contains the complete package notices and preserved bundler legal comments. Keep these attribution texts with redistributed runtime copies.

## Complete runtime package inventory

“Notice” links identify the exact license text below. Only byte-identical notice files are deduplicated; different copyright statements remain separate.

| Package | Version | Terms | Notice | Official package archive |
| --- | --- | --- | --- | --- |
| `parse5` | `8.0.0` | MIT | [Notice 27](#notice-27) | [npm tarball](https://registry.npmjs.org/parse5/-/parse5-8.0.0.tgz) |
| `entities` | `6.0.1` | BSD-2-Clause | [Notice 28](#notice-28) | [npm tarball](https://registry.npmjs.org/entities/-/entities-6.0.1.tgz) |
| `@openuidev/lang-core` | `1.0.0` | MIT | [Notice 1](#notice-1) | [npm tarball](https://registry.npmjs.org/@openuidev/lang-core/-/lang-core-1.0.0.tgz) |
| `@openuidev/observability` | `0.0.4` | MIT | [Notice 1](#notice-1) | [npm tarball](https://registry.npmjs.org/@openuidev/observability/-/observability-0.0.4.tgz) |
| `@openuidev/react-lang` | `1.0.0` | MIT | [Notice 1](#notice-1) | [npm tarball](https://registry.npmjs.org/@openuidev/react-lang/-/react-lang-1.0.0.tgz) |
| `@ungap/structured-clone` | `1.4.0` | ISC | [Notice 2](#notice-2) | [npm tarball](https://registry.npmjs.org/@ungap/structured-clone/-/structured-clone-1.4.0.tgz) |
| `bail` | `2.0.2` | MIT | [Notice 3](#notice-3) | [npm tarball](https://registry.npmjs.org/bail/-/bail-2.0.2.tgz) |
| `ccount` | `2.0.1` | MIT | [Notice 3](#notice-3) | [npm tarball](https://registry.npmjs.org/ccount/-/ccount-2.0.1.tgz) |
| `comma-separated-tokens` | `2.0.3` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/comma-separated-tokens/-/comma-separated-tokens-2.0.3.tgz) |
| `decode-named-character-reference` | `1.3.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/decode-named-character-reference/-/decode-named-character-reference-1.3.0.tgz) |
| `escape-string-regexp` | `5.0.0` | MIT | [Notice 6](#notice-6) | [npm tarball](https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-5.0.0.tgz) |
| `estree-util-is-identifier-name` | `3.0.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/estree-util-is-identifier-name/-/estree-util-is-identifier-name-3.0.0.tgz) |
| `extend` | `3.0.2` | MIT | [Notice 8](#notice-8) | [npm tarball](https://registry.npmjs.org/extend/-/extend-3.0.2.tgz) |
| `hast-util-to-jsx-runtime` | `2.3.6` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/hast-util-to-jsx-runtime/-/hast-util-to-jsx-runtime-2.3.6.tgz) |
| `hast-util-whitespace` | `3.0.0` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/hast-util-whitespace/-/hast-util-whitespace-3.0.0.tgz) |
| `highlight.js` | `11.11.1` | BSD-3-Clause | [Notice 26](#notice-26) | [npm tarball](https://registry.npmjs.org/highlight.js/-/highlight.js-11.11.1.tgz) |
| `html-url-attributes` | `3.0.1` | MIT | [Notice 9](#notice-9) | [npm tarball](https://registry.npmjs.org/html-url-attributes/-/html-url-attributes-3.0.1.tgz) |
| `inline-style-parser` | `0.2.7` | MIT | [Notice 10](#notice-10) | [npm tarball](https://registry.npmjs.org/inline-style-parser/-/inline-style-parser-0.2.7.tgz) |
| `is-plain-obj` | `4.1.0` | MIT | [Notice 6](#notice-6) | [npm tarball](https://registry.npmjs.org/is-plain-obj/-/is-plain-obj-4.1.0.tgz) |
| `katex` | `0.16.47` | MIT | [Notice 11](#notice-11) | [npm tarball](https://registry.npmjs.org/katex/-/katex-0.16.47.tgz) |
| `longest-streak` | `3.1.0` | MIT | [Notice 12](#notice-12) | [npm tarball](https://registry.npmjs.org/longest-streak/-/longest-streak-3.1.0.tgz) |
| `lowlight` | `3.3.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/lowlight/-/lowlight-3.3.0.tgz) |
| `lucide-react` | `0.575.0` | ISC + MIT (Feather portions) | [Notice 13](#notice-13) | [npm tarball](https://registry.npmjs.org/lucide-react/-/lucide-react-0.575.0.tgz) |
| `markdown-table` | `3.0.4` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/markdown-table/-/markdown-table-3.0.4.tgz) |
| `mdast-util-find-and-replace` | `3.0.3` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-find-and-replace/-/mdast-util-find-and-replace-3.0.3.tgz) |
| `mdast-util-from-markdown` | `2.1.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-from-markdown/-/mdast-util-from-markdown-2.1.0.tgz) |
| `mdast-util-gfm` | `3.1.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm/-/mdast-util-gfm-3.1.0.tgz) |
| `mdast-util-gfm-autolink-literal` | `2.0.1` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm-autolink-literal/-/mdast-util-gfm-autolink-literal-2.0.1.tgz) |
| `mdast-util-gfm-footnote` | `2.1.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm-footnote/-/mdast-util-gfm-footnote-2.1.0.tgz) |
| `mdast-util-gfm-strikethrough` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm-strikethrough/-/mdast-util-gfm-strikethrough-2.0.1.tgz) |
| `mdast-util-gfm-table` | `2.0.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm-table/-/mdast-util-gfm-table-2.0.0.tgz) |
| `mdast-util-gfm-task-list-item` | `2.0.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/mdast-util-gfm-task-list-item/-/mdast-util-gfm-task-list-item-2.0.0.tgz) |
| `mdast-util-phrasing` | `4.1.0` | MIT | [Notice 14](#notice-14) | [npm tarball](https://registry.npmjs.org/mdast-util-phrasing/-/mdast-util-phrasing-4.1.0.tgz) |
| `mdast-util-to-hast` | `13.2.1` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/mdast-util-to-hast/-/mdast-util-to-hast-13.2.1.tgz) |
| `mdast-util-to-markdown` | `2.2.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/mdast-util-to-markdown/-/mdast-util-to-markdown-2.2.0.tgz) |
| `mdast-util-to-string` | `4.0.0` | MIT | [Notice 3](#notice-3) | [npm tarball](https://registry.npmjs.org/mdast-util-to-string/-/mdast-util-to-string-4.0.0.tgz) |
| `micromark` | `4.0.3` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark/-/micromark-4.0.3.tgz) |
| `micromark-core-commonmark` | `2.0.4` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-core-commonmark/-/micromark-core-commonmark-2.0.4.tgz) |
| `micromark-extension-gfm` | `3.0.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm/-/micromark-extension-gfm-3.0.0.tgz) |
| `micromark-extension-gfm-autolink-literal` | `2.1.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm-autolink-literal/-/micromark-extension-gfm-autolink-literal-2.1.0.tgz) |
| `micromark-extension-gfm-footnote` | `2.1.0` | MIT | [Notice 15](#notice-15) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm-footnote/-/micromark-extension-gfm-footnote-2.1.0.tgz) |
| `micromark-extension-gfm-strikethrough` | `2.1.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm-strikethrough/-/micromark-extension-gfm-strikethrough-2.1.0.tgz) |
| `micromark-extension-gfm-table` | `2.1.2` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm-table/-/micromark-extension-gfm-table-2.1.2.tgz) |
| `micromark-extension-gfm-task-list-item` | `2.1.0` | MIT | [Notice 7](#notice-7) | [npm tarball](https://registry.npmjs.org/micromark-extension-gfm-task-list-item/-/micromark-extension-gfm-task-list-item-2.1.0.tgz) |
| `micromark-factory-destination` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-factory-destination/-/micromark-factory-destination-2.0.1.tgz) |
| `micromark-factory-label` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-factory-label/-/micromark-factory-label-2.0.1.tgz) |
| `micromark-factory-space` | `2.1.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-factory-space/-/micromark-factory-space-2.1.0.tgz) |
| `micromark-factory-title` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-factory-title/-/micromark-factory-title-2.0.1.tgz) |
| `micromark-factory-whitespace` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-factory-whitespace/-/micromark-factory-whitespace-2.0.1.tgz) |
| `micromark-util-character` | `2.1.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-character/-/micromark-util-character-2.1.1.tgz) |
| `micromark-util-chunked` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-chunked/-/micromark-util-chunked-2.0.1.tgz) |
| `micromark-util-classify-character` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-classify-character/-/micromark-util-classify-character-2.0.1.tgz) |
| `micromark-util-combine-extensions` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-combine-extensions/-/micromark-util-combine-extensions-2.0.1.tgz) |
| `micromark-util-decode-numeric-character-reference` | `2.0.2` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-decode-numeric-character-reference/-/micromark-util-decode-numeric-character-reference-2.0.2.tgz) |
| `micromark-util-decode-string` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-decode-string/-/micromark-util-decode-string-2.0.1.tgz) |
| `micromark-util-edit-map` | `1.0.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-edit-map/-/micromark-util-edit-map-1.0.0.tgz) |
| `micromark-util-html-tag-name` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-html-tag-name/-/micromark-util-html-tag-name-2.0.1.tgz) |
| `micromark-util-normalize-identifier` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-normalize-identifier/-/micromark-util-normalize-identifier-2.0.1.tgz) |
| `micromark-util-resolve-all` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-resolve-all/-/micromark-util-resolve-all-2.0.1.tgz) |
| `micromark-util-sanitize-uri` | `2.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-sanitize-uri/-/micromark-util-sanitize-uri-2.0.1.tgz) |
| `micromark-util-subtokenize` | `2.1.0` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/micromark-util-subtokenize/-/micromark-util-subtokenize-2.1.0.tgz) |
| `property-information` | `7.2.0` | MIT | [Notice 16](#notice-16) | [npm tarball](https://registry.npmjs.org/property-information/-/property-information-7.2.0.tgz) |
| `react` | `19.2.3` | MIT | [Notice 17](#notice-17) | [npm tarball](https://registry.npmjs.org/react/-/react-19.2.3.tgz) |
| `react-dom` | `19.2.3` | MIT | [Notice 17](#notice-17) | [npm tarball](https://registry.npmjs.org/react-dom/-/react-dom-19.2.3.tgz) |
| `react-markdown` | `10.1.0` | MIT | [Notice 18](#notice-18) | [npm tarball](https://registry.npmjs.org/react-markdown/-/react-markdown-10.1.0.tgz) |
| `remark-gfm` | `4.0.1` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/remark-gfm/-/remark-gfm-4.0.1.tgz) |
| `remark-parse` | `11.0.0` | MIT | [Notice 19](#notice-19) | [npm tarball](https://registry.npmjs.org/remark-parse/-/remark-parse-11.0.0.tgz) |
| `remark-rehype` | `11.1.2` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/remark-rehype/-/remark-rehype-11.1.2.tgz) |
| `scheduler` | `0.27.0` | MIT | [Notice 17](#notice-17) | [npm tarball](https://registry.npmjs.org/scheduler/-/scheduler-0.27.0.tgz) |
| `space-separated-tokens` | `2.0.2` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/space-separated-tokens/-/space-separated-tokens-2.0.2.tgz) |
| `style-to-js` | `1.1.21` | MIT | [Notice 20](#notice-20) | [npm tarball](https://registry.npmjs.org/style-to-js/-/style-to-js-1.1.21.tgz) |
| `style-to-object` | `1.0.14` | MIT | [Notice 21](#notice-21) | [npm tarball](https://registry.npmjs.org/style-to-object/-/style-to-object-1.0.14.tgz) |
| `trim-lines` | `3.0.1` | MIT | [Notice 12](#notice-12) | [npm tarball](https://registry.npmjs.org/trim-lines/-/trim-lines-3.0.1.tgz) |
| `trough` | `2.2.0` | MIT | [Notice 22](#notice-22) | [npm tarball](https://registry.npmjs.org/trough/-/trough-2.2.0.tgz) |
| `unified` | `11.0.5` | MIT | [Notice 23](#notice-23) | [npm tarball](https://registry.npmjs.org/unified/-/unified-11.0.5.tgz) |
| `unist-util-is` | `6.0.1` | MIT | [Notice 24](#notice-24) | [npm tarball](https://registry.npmjs.org/unist-util-is/-/unist-util-is-6.0.1.tgz) |
| `unist-util-position` | `5.0.0` | MIT | [Notice 3](#notice-3) | [npm tarball](https://registry.npmjs.org/unist-util-position/-/unist-util-position-5.0.0.tgz) |
| `unist-util-stringify-position` | `4.0.0` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/unist-util-stringify-position/-/unist-util-stringify-position-4.0.0.tgz) |
| `unist-util-visit` | `5.1.0` | MIT | [Notice 3](#notice-3) | [npm tarball](https://registry.npmjs.org/unist-util-visit/-/unist-util-visit-5.1.0.tgz) |
| `unist-util-visit-parents` | `6.0.2` | MIT | [Notice 4](#notice-4) | [npm tarball](https://registry.npmjs.org/unist-util-visit-parents/-/unist-util-visit-parents-6.0.2.tgz) |
| `vfile` | `6.0.3` | MIT | [Notice 23](#notice-23) | [npm tarball](https://registry.npmjs.org/vfile/-/vfile-6.0.3.tgz) |
| `vfile-message` | `4.0.3` | MIT | [Notice 5](#notice-5) | [npm tarball](https://registry.npmjs.org/vfile-message/-/vfile-message-4.0.3.tgz) |
| `zod` | `4.6.7` | MIT | [Notice 25](#notice-25) | [npm tarball](https://registry.npmjs.org/zod/-/zod-4.6.7.tgz) |

## Complete published license texts

The following 26 distinct notice texts are copied from the installed official packages. The SHA-256 values identify the original license-file bytes. Package associations preserve every contributing package’s copyright and permission notice.

### Notice 1

Applies to: `@openuidev/lang-core@1.0.0`, `@openuidev/observability@0.0.4`, `@openuidev/react-lang@1.0.0`.

Original license-file SHA-256: `b6bdd2d3d161722fb3f3afc039114cfa9f3236f036e1c98e0853bc3a85f6cd9e`.

```text
The MIT License (MIT)

Copyright (c) 2011-2024 Thesys Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 2

Applies to: `@ungap/structured-clone@1.4.0`.

Original license-file SHA-256: `dc6d4961d8b6ee747231582ae9c53ce1d66bf76bc9f5a28f554c0e97210953bf`.

```text
ISC License

Copyright (c) 2021, Andrea Giammarchi, @WebReflection

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE
OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.
```

### Notice 3

Applies to: `bail@2.0.2`, `ccount@2.0.1`, `mdast-util-to-string@4.0.0`, `unist-util-position@5.0.0`, `unist-util-visit@5.1.0`.

Original license-file SHA-256: `63cb98b3f6abfb3c3592c16f88253c1bdc834087bf52671e8ce5609e4eb693cf`.

```text
(The MIT License)

Copyright (c) 2015 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 4

Applies to: `comma-separated-tokens@2.0.3`, `hast-util-whitespace@3.0.0`, `mdast-util-to-hast@13.2.1`, `space-separated-tokens@2.0.2`, `unist-util-stringify-position@4.0.0`, `unist-util-visit-parents@6.0.2`.

Original license-file SHA-256: `ca4662cb5d1b738fbe5350c0d5485ba11773b4b7208974082ae6e129a52d631d`.

```text
(The MIT License)

Copyright (c) 2016 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 5

Applies to: `decode-named-character-reference@1.3.0`, `hast-util-to-jsx-runtime@2.3.6`, `markdown-table@3.0.4`, `mdast-util-find-and-replace@3.0.3`, `mdast-util-from-markdown@2.1.0`, `mdast-util-gfm@3.1.0`, `mdast-util-gfm-footnote@2.1.0`, `mdast-util-gfm-strikethrough@2.0.1`, `mdast-util-to-markdown@2.2.0`, `micromark@4.0.3`, `micromark-core-commonmark@2.0.4`, `micromark-extension-gfm-table@2.1.2`, `micromark-factory-destination@2.0.1`, `micromark-factory-label@2.0.1`, `micromark-factory-space@2.1.0`, `micromark-factory-title@2.0.1`, `micromark-factory-whitespace@2.0.1`, `micromark-util-character@2.1.1`, `micromark-util-chunked@2.0.1`, `micromark-util-classify-character@2.0.1`, `micromark-util-combine-extensions@2.0.1`, `micromark-util-decode-numeric-character-reference@2.0.2`, `micromark-util-decode-string@2.0.1`, `micromark-util-edit-map@1.0.0`, `micromark-util-html-tag-name@2.0.1`, `micromark-util-normalize-identifier@2.0.1`, `micromark-util-resolve-all@2.0.1`, `micromark-util-sanitize-uri@2.0.1`, `micromark-util-subtokenize@2.1.0`, `remark-gfm@4.0.1`, `remark-rehype@11.1.2`, `vfile-message@4.0.3`, `lowlight@3.3.0`.

Original license-file SHA-256: `dd1081884a92952802f4803110a6bb543acea9a814c786d58605b4c1219b5ebb`.

```text
(The MIT License)

Copyright (c) Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 6

Applies to: `escape-string-regexp@5.0.0`, `is-plain-obj@4.1.0`.

Original license-file SHA-256: `5c932d88256b4ab958f64a856fa48e8bd1f55bc1d96b8149c65689e0c61789d3`.

```text
MIT License

Copyright (c) Sindre Sorhus <sindresorhus@gmail.com> (https://sindresorhus.com)

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 7

Applies to: `estree-util-is-identifier-name@3.0.0`, `mdast-util-gfm-autolink-literal@2.0.1`, `mdast-util-gfm-table@2.0.0`, `mdast-util-gfm-task-list-item@2.0.0`, `micromark-extension-gfm@3.0.0`, `micromark-extension-gfm-autolink-literal@2.1.0`, `micromark-extension-gfm-strikethrough@2.1.0`, `micromark-extension-gfm-task-list-item@2.1.0`.

Original license-file SHA-256: `536804b3adbff17dbbb9ed0d2bf9655b78f07640278581e37e3d7475282d5e61`.

```text
(The MIT License)

Copyright (c) 2020 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 8

Applies to: `extend@3.0.2`.

Original license-file SHA-256: `97af7428f93afbab9bf79cd09254da1357161794f4255eb1b4599f04dd2d2e7c`.

```text
The MIT License (MIT)

Copyright (c) 2014 Stefan Thomas

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 9

Applies to: `html-url-attributes@3.0.1`.

Original license-file SHA-256: `e18cf5173e928f808f0cd208097fee8fbec9d28fc0255213938e4c559e29ff58`.

```text
(The MIT License)

Copyright (c) Titus Wormer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 10

Applies to: `inline-style-parser@0.2.7`.

Original license-file SHA-256: `395c95f20598da653ff1eb1cb9d74947c7fac689697147f1d192fd0242a0e635`.

```text
(The MIT License)

Copyright (c) 2012 TJ Holowaychuk <tj@vision-media.ca>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the 'Software'), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 11

Applies to: `katex@0.16.47`.

Original license-file SHA-256: `766ccc1f306c885aa45542a9846bbd0a505b27a0374f146778171c2254ce18e3`.

```text
The MIT License (MIT)

Copyright (c) 2013-2020 Khan Academy and other contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Notice 12

Applies to: `longest-streak@3.1.0`, `trim-lines@3.0.1`.

Original license-file SHA-256: `9f084fac69d8cf1e6ce983ba5a3499c7695ed74a26ec625c38f0fd19fddd5e10`.

```text
(The MIT License)

Copyright (c) 2015 Titus Wormer <mailto:tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 13

Applies to: `lucide-react@0.575.0`.

Original license-file SHA-256: `668dcc52803480e0a026b31140a4cae668772663cd764e5991d252eef03f98db`.

```text
ISC License

Copyright (c) for portions of Lucide are held by Cole Bemis 2013-2026 as part of Feather (MIT). All other copyright (c) for Lucide are held by Lucide Contributors 2026.

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

---

The MIT License (MIT) (for portions derived from Feather)

Copyright (c) 2013-2026 Cole Bemis

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Notice 14

Applies to: `mdast-util-phrasing@4.1.0`.

Original license-file SHA-256: `b6d1a4e1831b177acc96f22a817d4f0f4c18419355badc3125754445fc69e3a7`.

```text
(The MIT License)

Copyright (c) 2017 Titus Wormer <tituswormer@gmail.com>
Copyright (c) 2017 Victor Felder <victor@draft.li>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 15

Applies to: `micromark-extension-gfm-footnote@2.1.0`.

Original license-file SHA-256: `807a88522a850092aece49486207c17c313c4d5708ac276524840fe1caa0b0fa`.

```text
(The MIT License)

Copyright (c) 2021 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 16

Applies to: `property-information@7.2.0`.

Original license-file SHA-256: `5d0698266086f217bb11ea0499d33453e401fe479fb9126ad04e625e63d3557b`.

```text
(The MIT License)

Copyright (c) Titus Wormer <mailto:tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 17

Applies to: `react@19.2.3`, `react-dom@19.2.3`, `scheduler@0.27.0`.

Original license-file SHA-256: `da6d3703ed11cbe42bd212c725957c98da23cbff1998c05fa4b3d976d1a58e93`.

```text
MIT License

Copyright (c) Meta Platforms, Inc. and affiliates.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Notice 18

Applies to: `react-markdown@10.1.0`.

Original license-file SHA-256: `f6196c64e144f9a6fa9154c3a80bc8b89615a9567934b83a8951879f06ba2aef`.

```text
The MIT License (MIT)

Copyright (c) Espen Hovlandsdal

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Notice 19

Applies to: `remark-parse@11.0.0`.

Original license-file SHA-256: `413654b0a78b98661dce96d2aff2e4d075603d220b8483e28b1aa010ccc138b4`.

```text
(The MIT License)

Copyright (c) 2014 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 20

Applies to: `style-to-js@1.1.21`.

Original license-file SHA-256: `bbb380e5c1a717079378d6ea4d33c8a5a81655d4612b749f882e34e46712aab5`.

```text
The MIT License (MIT)

Copyright (c) 2020 Menglin "Mark" Xu <mark@remarkablemark.org>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 21

Applies to: `style-to-object@1.0.14`.

Original license-file SHA-256: `896c9fd6b5fc48ac31800b9d29a6e3ffcb0c2d534ed077ebfa5c2fa388cc38b7`.

```text
The MIT License (MIT)

Copyright (c) 2017 Menglin "Mark" Xu <mark@remarkablemark.org>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 22

Applies to: `trough@2.2.0`.

Original license-file SHA-256: `6c03fd41cfd7c92d8aa8a2fa521b94b2683f059123281dcf921ddea9216b6254`.

```text
(The MIT License)

Copyright (c) 2016 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 23

Applies to: `unified@11.0.5`, `vfile@6.0.3`.

Original license-file SHA-256: `05811400116ed61f1a7693b78fe8ca6598c155a12f3978e609db4df648a4cb3d`.

```text
(The MIT License)

Copyright (c) 2015 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 24

Applies to: `unist-util-is@6.0.1`.

Original license-file SHA-256: `82974dbf2639d13edab95c32ed9cb6c0867ede272cd2e07ce47ce8548fe55c05`.

```text
(The MIT license)

Copyright (c) 2015 Titus Wormer <tituswormer@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### Notice 25

Applies to: `zod@4.6.7`.

Original license-file SHA-256: `3f1189b28e3866e0d979968d466b78f813f76827cfdca1fbb124cc0a5c8841f8`.

```text
MIT License

Copyright (c) 2025 Colin McDonnell

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Notice 26

Applies to: `highlight.js@11.11.1`.

Original license-file SHA-256: `6c081431591d9df696c82dc598fe1423765b8a299b200ed00b281afd0f64c490`.

```text
BSD 3-Clause License

Copyright (c) 2006, Ivan Sagalaev.
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### Notice 27

Applies to: `parse5@8.0.0`.

Original license-file SHA-256: `8c535800331e1e4439835555b3f9edc7fe9dee2fab0d8bbbd5a982e8b8343d4d`.

```text
Copyright (c) 2013-2019 Ivan Nikulin (ifaaan@gmail.com, https://github.com/inikulin)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

### Notice 28

Applies to: `entities@6.0.1`.

Original license-file SHA-256: `cb992345949ccd6e8394b2cd6c465f7b897c864f845937dbf64e8997f389e164`.

```text
Copyright (c) Felix Böhm
All rights reserved.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS,
EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
