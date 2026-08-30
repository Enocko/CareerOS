package catalogreport

// ClassifyURLForTest exposes URL classification for unit tests.
func ClassifyURLForTest(url string) string {
	return classifyURL(url)
}
