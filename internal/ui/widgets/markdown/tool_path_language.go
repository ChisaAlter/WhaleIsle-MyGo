package markdown

import "strings"

// This table is the installed util/code-language languageForPath map. Diff
// banners compare canonical ids (JavaScript and TypeScript remain distinct).
var toolExtensionLanguages = func() map[string]string {
	groups := map[string]string{
		"typescript": "ts tsx mts cts", "javascript": "js jsx mjs cjs", "shellscript": "sh bash zsh", "fish": "fish", "json": "json jsonc jsonl ndjson ipynb", "csv": "csv",
		"python": "py pyw pyi", "ruby": "rb rake gemspec", "go": "go", "rust": "rs", "java": "java", "c": "c h", "cpp": "cc cpp cxx hh hpp hxx", "csharp": "cs",
		"kotlin": "kt kts", "swift": "swift", "php": "php", "yaml": "yaml yml", "toml": "toml", "ini": "ini conf cfg properties", "dotenv": "env", "log": "log", "diff": "diff patch", "http": "http",
		"markdown": "md markdown", "mdx": "mdx", "rst": "rst", "latex": "tex sty cls", "bibtex": "bib", "asciidoc": "adoc", "html": "html htm xhtml", "css": "css", "scss": "scss", "less": "less", "sql": "sql",
		"xml": "xml xsd xsl xslt plist svg", "lua": "lua", "bat": "bat cmd", "powershell": "ps1 psm1 psd1", "r": "r", "julia": "jl", "dart": "dart", "scala": "scala", "clojure": "clj cljs edn", "erlang": "erl hrl",
		"elixir": "ex exs", "haskell": "hs", "fsharp": "fs fsi fsx", "vb": "vb", "perl": "pl pm", "verilog": "v", "system-verilog": "sv svh", "graphql": "graphql gql", "proto": "proto", "hcl": "tf tfvars hcl",
		"nix": "nix", "vue": "vue", "svelte": "svelte", "make": "makefile mk", "cmake": "cmake", "groovy": "gradle groovy",
	}
	result := map[string]string{}
	for language, extensions := range groups {
		for _, extension := range strings.Fields(extensions) {
			result[extension] = language
		}
	}
	return result
}()

func toolLanguageForPath(path string) string {
	path = path[strings.LastIndexAny(path, `/\`)+1:]
	dot := strings.LastIndex(path, ".")
	if dot < 0 {
		return ""
	}
	return toolExtensionLanguages[strings.ToLower(path[dot+1:])]
}

// ToolDiffLanguage resolves the shared banner language only when all paths have
// the same installed canonical id. Missing/unknown hints return plain fallback.
func ToolDiffLanguage(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	first := toolLanguageForPath(paths[0])
	for _, path := range paths[1:] {
		if toolLanguageForPath(path) != first {
			return ""
		}
	}
	return first
}
