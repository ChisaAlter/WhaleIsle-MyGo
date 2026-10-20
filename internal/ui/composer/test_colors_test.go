package composer

import "whaleisle/internal/ui/theme"

// testColors is the app palette for view tests: the composer paints from
// theme.Colors, so tests pass the resolved light palette instead of one color.
func testColors() theme.Colors { return theme.Resolve("deepseek").Active(false) }
