package markdownsyntax

import (
	"reflect"
	"testing"
)

func TestImageReferencesKeepAllAuthoredOccurrences(t *testing.T) {
	source := "Before ![one](./plots/a_(1).png) and ![two](<C:/My plots/b (2).png> \"title\") after.\n\n![three](./My plots/c (3).png)\n\n![four][plot]\n\n[plot]: ./four.png\n\n[image] attachment:fixture\n\n![again](./four.png)"
	want := []string{"./plots/a_(1).png", "C:/My plots/b (2).png", "./My plots/c (3).png", "./four.png", "attachment:fixture", "./four.png"}
	if got := ImageReferences(source); !reflect.DeepEqual(got, want) {
		t.Fatalf("references=%q, want %q", got, want)
	}
}

func TestImageSyntaxIgnoresCodeAndEscapedExamples(t *testing.T) {
	source := "`![inline](inline.png)`\n\n```md\n![fenced](fenced.png)\n[image] attachment:fenced\n```\n\n\\![escaped](escaped.png)\n\n![real](./real_(1).png)"
	if got := ImageReferences(source); !reflect.DeepEqual(got, []string{"./real_(1).png"}) {
		t.Fatalf("code or escaped example became an attachment: %q", got)
	}
	want := "`![inline](inline.png)`\n\n```md\n![fenced](fenced.png)\n[image] attachment:fenced\n```\n\n\\![escaped](escaped.png)\n\n "
	if got := WithoutImages(source); got != want {
		t.Fatalf("prompt lost literal examples: %q, want %q", got, want)
	}
}

func TestWithoutImagesUsesCompletePathsAndTitles(t *testing.T) {
	source := "before ![one](a_(1).png) middle ![two](<./My pics/b (2).png> \"hello\") after"
	if got := WithoutImages(source); got != "before   middle   after" {
		t.Fatalf("image suffix leaked into prompt: %q", got)
	}
	refs, only := OnlyImages("![one](a_(1).png) ![two](b.png)")
	if !only || !reflect.DeepEqual(refs, []string{"a_(1).png", "b.png"}) {
		t.Fatalf("image-only projection=%q %v", refs, only)
	}
}
