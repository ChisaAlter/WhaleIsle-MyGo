package card

import (
	"context"

	"whaleisle/internal/desktop/media"
)

// pictureSize bounds the decoded size of card pictures (contain fit).
const pictureSize = 1600

// Image requests a public http(s) picture through the media service; the
// bytes come from the /api/image adapter (public addresses only, no
// credentials). It returns nil when the card has no image source.
func (c *Card) Image(src string) *media.Handle {
	o := c.mediaOwner()
	if o == nil {
		return nil
	}
	api := c.cfg.Resources
	key := media.Key{ID: "openui-image:" + src}.Sized(pictureSize, pictureSize)
	return c.cfg.Media.Request(o, key, media.Source{Load: func(ctx context.Context) ([]byte, error) { return api.PublicImage(ctx, src) }})
}

// SearchImage requests the first loadable picture the image-search adapter
// finds for query.
func (c *Card) SearchImage(query string) *media.Handle {
	o := c.mediaOwner()
	if o == nil {
		return nil
	}
	api := c.cfg.Resources
	key := media.Key{ID: "openui-search:" + query}.Sized(pictureSize, pictureSize)
	return c.cfg.Media.Request(o, key, media.Source{Load: func(ctx context.Context) ([]byte, error) { return api.SearchImage(ctx, query) }})
}

// mediaOwner is the card's media lease, released when the card closes.
func (c *Card) mediaOwner() *media.Owner {
	if c.cfg.Media == nil || c.cfg.Resources == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	if c.owner == nil {
		c.owner = c.cfg.Media.NewOwner(c.ctx, c.cfg.Invalidate)
	}
	return c.owner
}
