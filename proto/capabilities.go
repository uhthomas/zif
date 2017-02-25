package proto

func ChooseCompression(client MessageCapabilities, server MessageCapabilities) string {
	// check if the peer has our caps, in order of preference
	// the server has preference
	compression := ""
	for _, i := range server.Compression {
		if compression != "" {
			break
		}

		for _, j := range client.Compression {
			if i == j {
				compression = i
			}
		}
	}

	return compression
}
