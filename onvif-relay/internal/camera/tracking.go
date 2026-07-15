package camera

// SetTracking enables or disables the camera's motion tracking function.
func (c *Client) SetTracking(enabled bool) error {
	state := "off"
	if enabled {
		state = "on"
	}
	return c.SendCommand("property tracking " + state)
}
