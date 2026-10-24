package markdown

import "strings"

// The installed shared highlighter accepts this explicit alias table. Read
// cards with absent/unknown hints stay plain instead of guessing from content.
var toolGrammarAliases = func() map[string]string {
	groups := map[string]string{
		"typescript": "typescript ts tsx javascript js jsx", "shellscript": "shellscript bash sh shell zsh",
		"json": "json jsonc", "python": "py python", "ruby": "rb ruby", "go": "go", "rust": "rs rust",
		"java": "java", "c": "c", "cpp": "cpp", "csharp": "cs csharp", "kotlin": "kotlin", "swift": "swift", "php": "php",
		"yaml": "yaml yml", "toml": "toml", "ini": "ini properties", "markdown": "md markdown", "mdx": "mdx",
		"html": "html", "css": "css", "scss": "scss", "less": "less", "sql": "sql", "xml": "xml", "lua": "lua",
		"bat": "bat batch", "powershell": "powershell ps1 ps", "fish": "fish", "dotenv": "dotenv env", "log": "log", "csv": "csv",
		"diff": "diff patch", "http": "http", "rst": "rst", "latex": "latex tex", "bibtex": "bibtex bib", "asciidoc": "asciidoc adoc",
		"r": "r", "julia": "julia jl", "dart": "dart", "scala": "scala", "clojure": "clojure clj", "erlang": "erlang erl",
		"elixir": "elixir ex exs", "haskell": "haskell hs", "fsharp": "fsharp fs fsi fsx", "vb": "vb vbnet", "perl": "perl pl pm",
		"verilog": "verilog v", "system-verilog": "system-verilog systemverilog sv svh", "graphql": "graphql gql",
		"proto": "proto protobuf", "hcl": "hcl tf tfvars", "nix": "nix", "vue": "vue", "svelte": "svelte", "make": "make makefile mk",
		"cmake": "cmake", "groovy": "groovy gradle",
	}
	aliases := make(map[string]string)
	for grammar, names := range groups {
		for _, name := range strings.Fields(names) {
			aliases[name] = grammar
		}
	}
	return aliases
}()

func toolGrammarForHint(hint string) string { return toolGrammarAliases[strings.ToLower(hint)] }
