package git

func OriginRemotes(url string) []*Remote {
	return []*Remote{{Name: "origin", URL: url}}
}
