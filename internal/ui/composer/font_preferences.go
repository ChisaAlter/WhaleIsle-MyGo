package composer

func composerLineHeight(size float32) float32 { return size + 10 }

func (c *DraftClient) lineHeight() float32 { return composerLineHeight(c.font.Size) }
