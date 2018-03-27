package container

func UserNamespaceEnabled(c Client) (bool, error) {
	info, err := c.Info()
	if err != nil {
		return false, err
	}
	for _, val := range info.SecurityOptions {
		if val == "name=userns" {
			return true, nil
		}
	}
	return false, nil
}
