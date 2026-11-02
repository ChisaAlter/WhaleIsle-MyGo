package frame

// Chat chrome seats measured from the installed harness page at 1440x920, DIP.
//
// The values below close S27/S28: the composer card must be exactly as wide as
// the conversation body (774.4) instead of the old 772, the navigation label ink
// must sit on the native baseline (no 1.7-DIP offset), and the hero stack has a
// defined vertical seat instead of an implicit centring.
const (
	// conversationComposerClearance is InputBar.root's 16-DIP side clearance.
	// ConversationRoot centres axis+64 and each 16-DIP inset leaves the card at
	// axis+32, which is the native `max-width: 774.4px` body.
	conversationComposerClearance = float32(16)
	// conversationNavRowLineBox is the native panel row's 22-DIP label line box
	// under the row's 7-DIP padding at the default interface scale
	// (插件 label y=127, h=22). The sidebar paints theme.InterfaceSize(22), so a
	// user-scaled interface keeps the same proportion.
	conversationNavRowLineBox = float32(22)
	// conversationNavLabelCentre is the native label box's vertical centre
	// (7 padding + 11 half line box). A plain 18.7-DIP text box puts the ink
	// ~1.7 DIP lower, which was the whole S28 navigation residual.
	conversationNavLabelCentre = float32(138)
)

// conversationComposerBodyWidth is the native composer body for a shared
// content axis: the two 16-DIP InputBar insets around the axis
// (0.64 x 1160 + 32 = 774.4 at 1440x920). The composer card fills exactly this
// width, so composer and body cannot drift apart again.
func conversationComposerBodyWidth(contentAxis float32) float32 {
	return contentAxis + 2*conversationComposerClearance
}

// conversationComposerStackWidth is the centred stack ConversationRoot paints
// around the body: the body plus the 16-DIP clearance on each side.
func conversationComposerStackWidth(contentAxis float32) float32 {
	return conversationComposerBodyWidth(contentAxis) + 2*conversationComposerClearance
}
