package vault

func (c *Core) SendInvalidationNotice(keys ...string) {
	// Ensure we're called on the active node only.
	if c.standby.Load() {
		return
	}
}
